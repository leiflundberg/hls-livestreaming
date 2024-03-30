package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/gorilla/mux"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("chunk")
	if err != nil {
		http.Error(w, "Unable to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Unable to read file content", http.StatusInternalServerError)
		return
	}
	fmt.Println("Receiving file: ", handler.Filename)

	filepath := path.Join("webm", handler.Filename)

	err = os.WriteFile(filepath, fileBytes, 0644)
	if err != nil {
		http.Error(w, "Unable to write file to disk", http.StatusInternalServerError)
		return
	}

	outputDir := "hls/"
	playlistName := "output.m3u8"

	fmt.Println("Running ffmpeg conversion...")

	cmd := exec.Command("ffmpeg",
		"-i", filepath,
		"-c:v", "h264",
		"-f", "hls",
		"-crf", "30", // Adjust the CRF value for faster encoding (lower quality)
		"-preset", "ultrafast", // Use a faster encoding preset
		"-g", "48", // Set keyframe interval (lower values might increase quality but also increase file size)
		"-sc_threshold", "0", // Disable scene detection to speed up encoding
		"-f", "hls",
		"-hls_time", "4", // Set segment duration (lower values decrease latency but increase the number of files)
		"-hls_list_size", "0", // Do not limit the number of playlist entries
		"-hls_segment_filename", outputDir+"output%03d.ts",
		outputDir+playlistName,
	)

	err = cmd.Run()
	if err != nil {
		http.Error(w, "Failed to perform ffmpeg conversion", http.StatusInternalServerError)
		fmt.Println("Failed to perform ffmpeg conversion with error message: ", err.Error())
		return
	}
	fmt.Println("Ffmpeg conversion success!")
	fmt.Println("File sucessfully received and saved as: ", filepath)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File received: %s\n", handler.Filename)
}

// TODO: add errorhandling if exists etc...

func getIndex(w http.ResponseWriter, r *http.Request) {
	playlistPath := path.Join("hls", "output.m3u8")
	fmt.Println("Index file endpoint hit, serving: ", playlistPath)
	http.ServeFile(w, r, playlistPath)
}

func getSegment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	segmentName := vars["segment"]
	segmentPath := path.Join("hls", segmentName)
	fmt.Println("Segnment file endpoint hit, serving: ", segmentPath)
	http.ServeFile(w, r, segmentPath)
}

func main() {
	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.HandleFunc("/chunk", uploadHandler).Methods("POST")
	router.HandleFunc("/", getIndex).Methods("GET")
	router.HandleFunc("/{segment}", getSegment).Methods("GET")
	fmt.Println("Listening on port :8080...")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		handler.ServeHTTP(w, r)
	})
}
