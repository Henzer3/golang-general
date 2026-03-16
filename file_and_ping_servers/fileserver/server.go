package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"yadro.com/course/config"
)

const (
	UserFilesDir      = "userFiles"
	serverError       = "Server Error"
	fileNameParam     = "filename"
	fileFromMultipart = "file"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

func getPath(s string) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(pwd, UserFilesDir, s), nil
}

func closeFile(f io.Closer) {
	if err := f.Close(); err != nil {
		logger.Error("Close error", "err", err)
	}
}

func postFile(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile(fileFromMultipart)
	if err != nil {
		http.Error(w, "wrong multipart form", http.StatusBadRequest)
		return
	}
	defer closeFile(file)

	fileName := filepath.Base(header.Filename)
	filePath, err := getPath(fileName)
	if err != nil {
		logger.Error("getting path error", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
		return
	}

	if _, err := os.Stat(filePath); err == nil {
		w.WriteHeader(http.StatusConflict)
		return
	}

	dst, err := os.Create(filePath)
	if err != nil {
		logger.Error("cant create file", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
		return
	}
	defer closeFile(dst)

	if _, err = io.Copy(dst, file); err != nil {
		logger.Error("cant copy file", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if _, err := fmt.Fprintln(w, fileName); err != nil {
		logger.Error("cant write response", "err", err)
	}
}

func putFile(w http.ResponseWriter, r *http.Request) {
	fileName := r.PathValue(fileNameParam)
	filePath, err := getPath(fileName)

	if err != nil {
		logger.Error("getting path error", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
		return
	}

	// делаю через OpenFile чтобы сразу првоерить на существование файла и получение дискриптора для записи.
	// Если без OpenFile, то надо исопльзовать os.Create + os.Stat чтобы сделать и то и то.
	// Просто Open не получится так как он возвращает дискриптор только на чтение
	// 0666 в библиотеке os захардкожено, поэтому сомневаюсь делать ли константу для этого
	dst, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			logger.Error("cant open file", "err", err)
			http.Error(w, serverError, http.StatusInternalServerError)
		}
		return
	}
	defer closeFile(dst)

	file, _, err := r.FormFile(fileFromMultipart)
	if err != nil {
		http.Error(w, "wrong multipart form", http.StatusBadRequest)
		return
	}
	defer closeFile(file)

	if _, err := io.Copy(dst, file); err != nil {
		logger.Error("cant copy file", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getFiles(w http.ResponseWriter, r *http.Request) {
	dirPath, err := getPath("")
	if err != nil {
		logger.Error("getting path error", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
		return
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		logger.Error("Reading Directory error", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	for _, v := range entries {
		if _, err := fmt.Fprintln(w, v.Name()); err != nil {
			logger.Error("cant write response", "err", err)
			return
		}
	}
}

func getFile(w http.ResponseWriter, r *http.Request) {
	fileName := r.PathValue(fileNameParam)
	filePath, err := getPath(fileName)
	if err != nil {
		logger.Error("getting path error", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			logger.Error("Open file error", "err", err)
			http.Error(w, serverError, http.StatusInternalServerError)
		}
		return
	}
	defer closeFile(file)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err = io.Copy(w, file); err != nil {
		logger.Error("copying file error", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
	}
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	fileName := r.PathValue(fileNameParam)
	filePath, err := getPath(fileName)
	if err != nil {
		logger.Error("getteing path error", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
		return
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		logger.Error("error during os.Remove", "err", err)
		http.Error(w, serverError, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger.Error("cleanenv error", "err", err)
		return
	}

	port := ":" + cfg.Port

	directoryPath, err := getPath("")
	if err != nil {
		logger.Error("cant get pwd", "err", err)
		return
	}

	if info, err := os.Stat(directoryPath); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(directoryPath, 0755); err != nil {
				logger.Error("cant create directory", "err", err)
				return
			}
		} else {
			logger.Error("erroe during getting Stat", "err", err)
			return
		}
	} else if !info.IsDir() {
		logger.Error("cant create directory, exist file with this name", "name", UserFilesDir, "err", err)
		return
	}

	srv := http.NewServeMux()
	srv.HandleFunc("POST /files", postFile)
	srv.HandleFunc("PUT /files/{filename}", putFile)
	srv.HandleFunc("GET /files", getFiles)
	srv.HandleFunc("GET /files/{filename}", getFile)
	srv.HandleFunc("DELETE /files/{filename}", deleteFile)

	if err := http.ListenAndServe(port, srv); err != http.ErrServerClosed {
		logger.Error("error during ListenAndServe", "err", err)
		return
	}
}
