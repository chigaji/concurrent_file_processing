# concurrent_file_processing

Given multiple large files, count the occurrence of a given word concurrently across all files

## How to Use

### 1. Load the configuration

create a `config.yaml` and fill in your variables
samples :

```yaml  
    files:
        - "./file1.txt"
        - "./file2.txt"
        - "./file3.txt"
    word : "the"
    workerCount: 5
```

OR set evironment variables directly in the terminal

```bash
    export APP_FILES="./file1.txt,./file2.txt,./sample.txt"
    export APP_WORD="journey" 
    export APP_WORKERCOUNT=3
```

### 2. Run the program

``` bash
    go run main.go
```
