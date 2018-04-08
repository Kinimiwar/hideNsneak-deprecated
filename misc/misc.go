package misc

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

//misc.WriteActivityLog writes general activity to log file
func WriteActivityLog(text string) {
	//TODO Finalize a path for the activity log
	f, err := os.OpenFile("/tmp/hideNsneak/activity.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("Error opening activity log file, no log will be written: %s", err)
	}

	defer f.Close()

	if _, err = f.WriteString(time.Now().UTC().Format(time.RFC850) + " : " + text + "\n"); err != nil {
		log.Printf("Error writing activity log file, no log will be written: %s \n", err)
	}
}

//misc.WriteErrorLog writes application errors to log file
func WriteErrorLog(text string) bool {
	//TODO Finalize a path for the error log
	f, err := os.OpenFile("/tmp/hideNsneak/error.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("Error opening error log file, no log will be written: %s \n", err)
		return false
	}

	defer f.Close()

	if _, err = f.WriteString(time.Now().UTC().Format(time.RFC850) + " : " + text + "\n"); err != nil {
		log.Printf("Error writing error log file, no log will be written: %s \n", err)
		return false
	}
	return true
}

///////////////////

//String Slice Helper Functions//
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func RemoveString(s []string, e string) []string {
	for i := range s {
		if s[i] == e {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func RemoveDuplicateStrings(inSlice []string) (outSlice []string) {
	outSlice = inSlice[:1]
	for _, p := range inSlice {
		inOutSlice := false
		for _, q := range outSlice {
			if p == q {
				inOutSlice = true
			}
		}
		if !inOutSlice {
			outSlice = append(outSlice, p)
		}
	}
	return
}

func SplitOnComma(inString string) (outSlice []string) {
	outSlice = strings.Split(inString, ",")
	return
}

func ValidateIntArray(integers []string) bool {
	for _, p := range integers {
		_, err := strconv.Atoi(p)
		if err != nil {
			return false
		}
	}
	return true
}