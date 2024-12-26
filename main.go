package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
	case "ls-tree":
		command, _ := os.Args[2], os.Args[3]
		if command != "--name-only" {
			break
		}
		filenames := []string{}
		files, err := os.ReadDir(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unknown error %s\n", err)
			os.Exit(1)
		}
		for _, file := range files {
			if file.Name() != ".git" {
				filenames = append(filenames, file.Name())
			}
		}
		sort.Strings(filenames)
		result := strings.Join(filenames, "\n") + "\n"
		fmt.Print(result)
		break
	case "write-tree":
		currentDir, _ := os.Getwd()
		h, c := calcTreeHash(currentDir)
		treeHash := hex.EncodeToString(h)
		os.Mkdir(filepath.Join(".git", "objects", treeHash[:2]), 0755)
		var compressed bytes.Buffer
		w := zlib.NewWriter(&compressed)
		w.Write(c)
		w.Close()
		os.WriteFile(filepath.Join(".git", "objects", treeHash[:2], treeHash[2:]), compressed.Bytes(), 0644)
		fmt.Println(treeHash)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %s\n", command)
		os.Exit(1)
	}
}

func calcTreeHash(dir string) ([]byte, []byte) {
	fileInfos, _ := os.ReadDir(dir)
	type entry struct {
		fileName string
		b        []byte
	}
	var entries []entry
	contentSize := 0
	for _, fileInfo := range fileInfos {
		if fileInfo.Name() == ".git" {
			continue
		}
		if !fileInfo.IsDir() {
			f, _ := os.Open(filepath.Join(dir, fileInfo.Name()))
			b, _ := io.ReadAll(f)
			s := fmt.Sprintf("blob %d\u0000%s", len(b), string(b))
			sha1 := sha1.New()
			io.WriteString(sha1, s)
			s = fmt.Sprintf("100644 %s\u0000", fileInfo.Name())
			b = append([]byte(s), sha1.Sum(nil)...)
			entries = append(entries, entry{fileInfo.Name(), b})
			contentSize += len(b)
		} else {
			b, _ := calcTreeHash(filepath.Join(dir, fileInfo.Name()))
			s := fmt.Sprintf("40000 %s\n0000", fileInfo.Name())
			b2 := append([]byte(s), b...)
			entries = append(entries, entry{fileInfo.Name(), b2})
			contentSize += len(b2)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].fileName < entries[j].fileName })
	s := fmt.Sprintf("tree %d\u0000", contentSize)
	b := []byte(s)
	for _, entry := range entries {
		b = append(b, entry.b...)
	}
	sha1 := sha1.New()
	io.WriteString(sha1, string(b))
	return sha1.Sum(nil), b
}
