package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Fprintf(os.Stderr, "Logs will appear here!\n")

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}
		headFileContents := []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}
		fmt.Println("Initialized git directory")
	case "cat-file":
		sha := os.Args[3]
		path := fmt.Sprintf(".git/object/%v/%v", sha[0:2], sha[2:])
		file, _ := os.Open(path)
		r, _ := zlib.NewReader(io.Reader(file))
		s, _ := io.ReadAll(r)
		parts := strings.Split(string(s), "\x00")
		fmt.Print(parts[1])
		r.Close()
	case "hash-object":
		file, _ := os.ReadFile(os.Args[3])
		stats, _ := os.Stat(os.Args[3])
		content := string(file)
		contentAndHeader := fmt.Sprintf("blob %d\x00%s", stats.Size(), content)
		sha := (sha1.Sum([]byte(contentAndHeader)))
		hash := fmt.Sprintf("%x", sha)
		blobName := []rune(hash)
		blobPath := ".git/objects/"
		for i, v := range blobName {
			blobPath += string(v)
			if i == 1 {
				blobPath += "/"
			}
		}
		var buffer bytes.Buffer
		z := zlib.NewWriter(&buffer)
		z.Write([]byte(contentAndHeader))
		z.Close()
		os.MkdirAll(filepath.Dir(blobPath), os.ModePerm)
		f, _ := os.Create(blobPath)
		defer f.Close()
		f.Write(buffer.Bytes())
		fmt.Print(hash)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %s\n", command)
		os.Exit(1)
	}
}
