package main

import "fmt"

// OriginalSize - keep original size of image
const OriginalSize = "o"

// ImagePath computes image path based on its name
type ImagePath struct {
	rootDir string
}

// ImagePath - string
func (i *ImagePath) ImagePath(sz, name, ext string) string {
	if sz == OriginalSize {
		return fmt.Sprintf("%s/%s/%s", i.rootDir, sz, name) // сохраняем оригинальный размер без расширения
	}
	return fmt.Sprintf("%s/%s/%s%s", i.rootDir, sz, name, ext)
}
