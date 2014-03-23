// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package image contains methods for manipulating images, including
// resizing, cropping etc.
package image

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/nfnt/resize"
	"github.com/thoughtmonster/sleepy/core/config"
	"github.com/thoughtmonster/sleepy/core/server"
	"github.com/thoughtmonster/sleepy/core/user"
)

type Image struct {
	// Contains private or unexported fields.
	conf *config.Config
	id   map[string]string
}

type Request struct {
	Auth     string
	Remote   string
	Checksum string
	Filename string
	W        int64
	H        int64
	X        int64
	Y        int64
	Aspect   float64
}

func (i *Image) Crop(p Request) (string, error) {
	datadir := i.conf.S("directories", "data")
	address := i.conf.S("http", "address")
	port := i.conf.S("http", "port")

	path, err := i.filepath(fmt.Sprintf("op=crop;x=%d;y=%d;w=%d;h=%d", p.X, p.Y, p.W, p.H), &p)
	if err != nil {
		return "", nil
	}

	// Check for cached file.
	if _, err = os.Stat(datadir + "/serve" + path + p.Filename); err == nil {
		os.Remove(os.TempDir() + "/sleepy/" + i.id[p.Auth] + "/" + p.Checksum)
		return address + ":" + port + path + p.Filename, nil
	}

	// Upload and process image.
	img, format, err := i.upload(&p)
	if err != nil {
		return "", nil
	}

	b := img.Bounds()
	maxX := int(p.X + p.W)
	maxY := int(p.Y + p.H)

	// Crop only if the sub-image is inside the bounds of the source image.
	if b.Max.X >= maxX && b.Max.Y >= maxY {
		m := image.NewRGBA(b)
		draw.Draw(m, b, img, b.Min, draw.Src)

		t := m.SubImage(image.Rect(int(p.X), int(p.Y), maxX, maxY))

		err = generate(t, format, datadir+"/serve"+path, p.Filename)
		if err != nil {
			return "", nil
		}

		return address + ":" + port + path + p.Filename, nil
	}

	return "", nil
}

func (i *Image) Resize(p Request) (string, error) {
	datadir, _ := i.conf.String("directories", "data")
	address, _ := i.conf.String("http", "address")
	port, _ := i.conf.String("http", "port")

	path, err := i.filepath(fmt.Sprintf("op=resize;w=%d;h=%d;a=%f", p.W, p.H, p.Aspect), &p)
	if err != nil {
		return "", nil
	}

	// Check for cached file.
	if _, err = os.Stat(datadir + "/serve" + path + p.Filename); err == nil {
		os.Remove(os.TempDir() + "/sleepy/" + i.id[p.Auth] + "/" + p.Checksum)
		return address + ":" + port + path + p.Filename, nil
	}

	// Upload and process image.
	img, format, err := i.upload(&p)
	if err != nil {
		return "", nil
	}

	var t image.Image
	b := img.Bounds()

	if p.Aspect == 0 {
		factor := math.Min(float64(p.W)/float64(b.Max.X), float64(p.H)/float64(b.Max.Y))
		t = resize.Resize(uint(factor*float64(b.Max.X)), uint(factor*float64(b.Max.Y)), img, resize.Bicubic)
	} else {
		t = resize.Resize(uint(p.W), uint(p.H), img, resize.Bicubic)
	}

	err = generate(t, format, datadir+"/serve"+path, p.Filename)
	if err != nil {
		return "", nil
	}

	return address + ":" + port + path + p.Filename, nil
}

func (i *Image) filepath(options string, p *Request) (string, error) {
	if len(p.Checksum) != 40 {
		return "", fmt.Errorf("checksum does not appear to be an SHA-1 hash.")
	}

	if _, exists := i.id[p.Auth]; !exists {
		u, err := user.Auth(p.Auth)
		if err != nil {
			return "", err
		}

		i.id[p.Auth] = strconv.FormatInt(int64(u.Id), 10)
	}

	c := p.Checksum
	hash := c[:2] + "/" + c[2:6] + "/" + c[6:14] + "/" + c[14:27] + "/" + c[27:]

	opts := base64.StdEncoding.EncodeToString([]byte(options))
	path := "/" + i.id[p.Auth] + "/" + hash + "/" + opts + "/"

	return path, nil
}

func (i *Image) upload(p *Request) (image.Image, string, error) {
	var err error
	var src io.ReadCloser

	if p.Remote != "" {
		resp, err := http.Get(p.Remote)
		if err != nil {
			return nil, "", err
		} else if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, "", fmt.Errorf("receiving image failed with code: %d", resp.StatusCode)
		}

		src = resp.Body
	} else {
		tmpfile := os.TempDir() + "/sleepy/" + i.id[p.Auth] + "/" + p.Checksum
		if src, err = os.Open(tmpfile); err != nil {
			return nil, "", err
		}

		defer os.Remove(tmpfile)
	}

	defer src.Close()

	img, format, err := image.Decode(src)
	if err != nil {
		return nil, "", fmt.Errorf("image could not be decoded: %s", err)
	}

	return img, format, nil
}

func generate(img image.Image, format, path, filename string) error {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	out, err := os.Create(path + filename)
	if err != nil {
		return err
	}

	defer out.Close()

	switch format {
	case "jpeg":
		jpeg.Encode(out, img, &jpeg.Options{90})
	case "png":
		png.Encode(out, img)
	}

	return nil
}

func (i *Image) Setup(config *config.Config) error {
	i.conf = config

	return nil
}

func init() {
	server.Register(&Image{
		&config.Config{},
		make(map[string]string),
	})
}
