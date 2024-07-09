package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func visit(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if !info.IsDir() {
		err := copyFile(path, "temp"+path)
		if err != nil {
			fmt.Printf("Error copying file %s: %v\n", path, err)
		} else {
			fmt.Printf("Copied file %s to temp%s\n", path, path)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create all directories in path to dst
	err = os.MkdirAll(filepath.Dir(dst), os.ModePerm)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run copy_files.go <directory>")
		return
	}
	root := os.Args[1]

	err := filepath.Walk(root, visit)
	if err != nil {
		fmt.Printf("Error walking the path %s: %v\n", root, err)
	}
}
