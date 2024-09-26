package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// process a unit of work for each file
type Job struct {
	FilePath string
	Word     string
}

// Result holds the result of processing a file
type Result struct {
	FilePath  string
	WordCount int
	Error     error
}

type FileProcessor struct {
	Files       []string    // the file paths to process
	Word        string      // the word to look for
	Results     chan Result // channel to store results
	WorkerCount int         // the number of workers to use
}

// Initialize the file processor
func NewFileProcessor(files []string, word string, workerCount int) *FileProcessor {
	return &FileProcessor{
		Files:       files,
		Word:        word,
		WorkerCount: workerCount,
	}
}

// Load configuration in config.yaml file
func LoadConfig() (*FileProcessor, error) {
	//  set up viper to read from the config.yaml file and environment variables
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// environment variable support with prefixes
	viper.AutomaticEnv()

	viper.SetEnvPrefix("app") // env variable prefix
	viper.BindEnv("files")    // binds APP_FILES to override the file path
	viper.BindEnv("word")     // binds APP_WORD to override search word
	viper.BindEnv("workerCount")

	// read in config file
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading in config file:  %s\n", err)
	}

	// set defaut values if not provided in config file or environment variable
	viper.SetDefault("files", []string{"./sample1.txt"})
	viper.SetDefault("word", "go")
	viper.SetDefault("workerCount", 1)

	// extract configuration values
	files := viper.GetStringSlice("files")
	word := viper.GetString("word")
	workerCount := viper.GetInt("workerCount")

	// handle the comma-separated list from env variable
	if envfiles := viper.GetString("files"); envfiles != "" {
		files = strings.Split(envfiles, ",")
	}

	fmt.Println("env files := ", files)
	fmt.Println("word := ", word)
	fmt.Println("worker := ", workerCount)

	// Iniatialize FileProcessor with the loaded config
	fp := NewFileProcessor(files, word, workerCount)
	return fp, nil
}

// initializes file processing
func (fp *FileProcessor) ProcessFiles(ctx context.Context) {
	jobs := make(chan Job, len(fp.Files))         // job channel
	fp.Results = make(chan Result, len(fp.Files)) // result collection channel

	// a waitgroup to syncronize all go routines
	var wg sync.WaitGroup

	// start the worker goroutines
	for i := 0; i < fp.WorkerCount; i++ {
		wg.Add(1)
		go fp.Worker(ctx, jobs, &wg)
	}

	// feed jobs to the workers
	for _, filepath := range fp.Files {
		jobs <- Job{FilePath: filepath, Word: fp.Word}
	}

	close(jobs) // close jobs channel since no more jobs are being added

	wg.Wait() // wait for all goroutines to finish

	close(fp.Results) // close the result channel to signal job completion
}

// process jobs concurrently
func (fp *FileProcessor) Worker(ctx context.Context, job <-chan Job, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case job, ok := <-job:
			if !ok {
				return // return if no more jobs
			}
			// process and send results
			result := fp.CountWord(ctx, job)
			fp.Results <- result
		case <-ctx.Done():
			fmt.Println("Shutting down gracefully due to context cancellation")
			return
		}
	}
}

// count the occurence of a word in a single file
func (fp *FileProcessor) CountWord(ctx context.Context, job Job) Result {
	file, err := os.Open(job.FilePath)

	if err != nil {
		return Result{FilePath: job.FilePath, Error: err}
	}

	defer file.Close()

	wordCount := 0
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return Result{FilePath: job.FilePath, Error: ctx.Err()}
		default:
			line := scanner.Text()
			wordCount += strings.Count(line, job.Word)
		}
	}

	if err := scanner.Err(); err != nil {
		return Result{FilePath: job.FilePath, Error: err}
	}

	return Result{FilePath: job.FilePath, WordCount: wordCount}

}

func main() {
	// filePaths := []string{
	// 	"./file1.txt",
	// 	"./file2.txt",
	// 	"./file3.txt",
	// }

	// fp := NewFileProcessor(filePaths, "from", 3)
	//load enviroment variable and initialize FileProcessor
	fp, err := LoadConfig()

	if err != nil {
		fmt.Printf("Error processing configuration : %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	//start file processing
	fp.ProcessFiles(ctx)

	//collect and print results

	for result := range fp.Results {
		if result.Error != nil {
			fmt.Printf("Error processing file %s:, %v\n", result.FilePath, result.Error)
		}
		fmt.Printf("Processed file: %s; Word Count: %d\n", result.FilePath, result.WordCount)
	}

}
