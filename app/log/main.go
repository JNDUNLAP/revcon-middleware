package log

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Level int

const (
	DEBUG Level = iota + 1
	INFO
	WARNING
	ERROR
	FATAL
)

const (
	colorRed            = "\033[91m"
	colorGreen          = "\033[92m"
	colorYellow         = "\033[93m"
	colorMagenta        = "\033[95m"
	colorCyan           = "\033[36m"
	colorWhite          = "\033[17m"
	colorReset          = "\033[0m"
	timeColor           = "\033[1;30m"
	colorShortDuration  = "\033[32m" // Green for short durations
	colorMediumDuration = "\033[33m" // Yellow for medium durations
	colorLongDuration   = "\033[31m" // Red for long durations
)

const (
	defaultShortThreshold  = 100 * time.Millisecond
	defaultMediumThreshold = 500 * time.Millisecond
)

var (
	shortDurationThreshold, mediumDurationThreshold time.Duration = defaultShortThreshold, defaultMediumThreshold
)

type DurationHistory struct {
	Durations []time.Duration
	Index     int
	Full      bool
}

func NewDurationHistory(size int) *DurationHistory {
	return &DurationHistory{
		Durations: make([]time.Duration, size),
		Index:     0,
		Full:      false,
	}
}

func (dh *DurationHistory) Add(duration time.Duration) {
	dh.Durations[dh.Index] = duration
	dh.Index++
	if dh.Index == len(dh.Durations) {
		dh.Index = 0
		dh.Full = true
	}
}

func (dh *DurationHistory) CalculateThresholds() (shortThreshold, mediumThreshold time.Duration) {
	if !dh.Full {
		return defaultShortThreshold, defaultMediumThreshold // Use default thresholds initially
	}

	var total time.Duration
	for _, d := range dh.Durations {
		total += d
	}
	average := total / time.Duration(len(dh.Durations))

	shortThreshold = average / 2
	mediumThreshold = average * 2

	return shortThreshold, mediumThreshold
}

func (dh *DurationHistory) ShouldRecalculate() bool {
	return dh.Full // Recalculate when the buffer is full
}

var levelStrings = []string{
	"DEBUG", "INFO", "WARNING", "ERROR", "FATAL",
}

var levelColors = map[string]string{
	"DEBUG":   colorCyan,
	"INFO":    colorGreen,
	"WARNING": colorYellow,
	"ERROR":   colorRed,
	"FATAL":   colorMagenta,
}

func (l Level) String() string {
	if l < DEBUG || l > FATAL {
		return "UNKNOWN"
	}
	return levelStrings[l-1]
}

type LogOutput interface {
	Write(message string) error
	Close() error
}

type ConsoleLogOutput struct{}

func (c *ConsoleLogOutput) Write(message string) error {
	return nil
}

func (c *ConsoleLogOutput) Close() error {
	return nil
}

type MongoDBLogOutput struct {
	client         *mongo.Client
	databaseName   string
	collectionName string
}

func NewMongoDBLogOutput(uri, databaseName, collectionName string) (*MongoDBLogOutput, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}

	return &MongoDBLogOutput{
		client:         client,
		databaseName:   databaseName,
		collectionName: collectionName,
	}, nil
}

func (m *MongoDBLogOutput) Write(logEntry string) error {
	parts := strings.SplitN(logEntry, " ", 4)
	if len(parts) < 4 {
		return fmt.Errorf("log message format error")
	}

	timestamp, level, message := parts[0], parts[1], parts[3]
	logDocument := bson.M{
		"timestamp": timestamp,
		"level":     level,
		"message":   message,
	}

	collection := m.client.Database(m.databaseName).Collection(m.collectionName)
	_, err := collection.InsertOne(context.Background(), logDocument)
	return err
}

func (m *MongoDBLogOutput) Close() error {
	return m.client.Disconnect(context.Background())
}

type CompositeLogOutput struct {
	outputs []LogOutput
}

func NewCompositeLogOutput(outputs ...LogOutput) *CompositeLogOutput {
	return &CompositeLogOutput{outputs: outputs}
}

func (c *CompositeLogOutput) Write(message string) error {
	var err error
	for _, output := range c.outputs {
		if e := output.Write(message); e != nil {
			err = e
		}
	}
	return err
}

func (c *CompositeLogOutput) Close() error {
	var err error
	for _, output := range c.outputs {
		if e := output.Close(); e != nil {
			err = e
		}
	}
	return err
}

type Logger struct {
	mu              sync.Mutex
	output          LogOutput
	level           Level
	timestampFormat string
	printLogs       bool
	durationHistory *DurationHistory
}

func NewLogger(level Level, output LogOutput, timestampFormat string, printLogs bool, historySize int) *Logger {
	return &Logger{
		output:          output,
		level:           level,
		timestampFormat: timestampFormat,
		printLogs:       printLogs,
		durationHistory: NewDurationHistory(historySize),
	}
}

func (l *Logger) SetConfig(level Level, output LogOutput, timestampFormat string, printLogs bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.level = level
	l.output = output
	l.timestampFormat = timestampFormat
	l.printLogs = printLogs
}

func GetCurrentFunctionName() string {
	pc, _, _, ok := runtime.Caller(3) // Use 2 to get the caller of the log function
	if !ok {
		return "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	return fn.Name()
}

func getDurationColor(duration time.Duration) string {
	switch {
	case duration <= shortDurationThreshold:
		return colorShortDuration
	case duration <= mediumDurationThreshold:
		return colorMediumDuration
	default:
		return colorLongDuration
	}
}

func (l *Logger) log(level Level, format string, v ...interface{}) {
	start := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	duration := time.Since(start)
	l.durationHistory.Add(duration)

	if l.durationHistory.ShouldRecalculate() {
		shortDurationThreshold, mediumDurationThreshold = l.durationHistory.CalculateThresholds()
	}

	durationColor := getDurationColor(duration)
	functionName := GetCurrentFunctionName()
	message := fmt.Sprintf(format, v...)
	timestamp := time.Now().Format(l.timestampFormat)
	logEntry := fmt.Sprintf("%s[%s]%s | %s%s%s | %s | %s | %s%v%s\n", timeColor, timestamp, colorReset, colorGreen, level, colorReset, message, functionName, durationColor, duration, colorReset)

	if l.printLogs {
		fmt.Print(logEntry)
	}

	if l.output != nil {
		l.output.Write(logEntry)
	}
}

func Info(format string, v ...interface{}) {
	globalLogger.log(INFO, format, v...)
}

func Debug(format string, v ...interface{}) {
	globalLogger.log(DEBUG, format, v...)
}

func Warning(format string, v ...interface{}) {
	globalLogger.log(WARNING, format, v...)
}

func Error(format string, v ...interface{}) {
	globalLogger.log(ERROR, format, v...)
}

func Fatal(format string, v ...interface{}) {
	globalLogger.log(FATAL, format, v...)
	os.Exit(1)
}

var globalLogger *Logger

func InitializeMongoDBLogger(printlogs bool, historySize int) {
	consoleOutput := &ConsoleLogOutput{}
	mongoDBOutput, err := NewMongoDBLogOutput(os.Getenv("MongoURI"), "honda", "revcon_api_logs")
	if err != nil {
		fmt.Println("Error connecting to database for logs:", err)
		return
	}

	compositeOutput := NewCompositeLogOutput(consoleOutput, mongoDBOutput)
	globalLogger = NewLogger(INFO, compositeOutput, time.RFC3339, printlogs, historySize)
}
