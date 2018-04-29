package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/aki237/dibba"
	"github.com/aki237/spacelang"
	"github.com/blackspace/gofb/framebuffer"
)

func getGetFileFromZip(buf *zip.ReadCloser, filename string) (io.Reader, error) {
	for _, f := range buf.File {
		if f.Name == filename {
			return f.Open()
		}
	}
	return nil, errors.New("file not found")
}

func getVariable(vm *spacelang.VM, a string) (interface{}, error) {
	v, ok := vm.Vars[a]
	if !ok {
		return nil, errors.New("variable not found")
	}
	return v, nil
}

func main() {
	if len(os.Args) != 2 {
		return
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	d := dibba.NewReader(f)
	err = d.Parse()
	if err != nil {
		fmt.Println(err)
		return
	}

	fb := framebuffer.NewFramebuffer()
	vm := spacelang.NewVM()
	vm.Funcs["clear"] = func(a ...*spacelang.Token) error {
		if len(a) != 0 {
			return errors.New("no args expected")
		}
		fb.Fill(0, 0, 0, 0)
		return nil
	}

	vm.Funcs["sleep"] = func(a ...*spacelang.Token) error {
		var duration int64

		if len(a) != 1 {
			return fmt.Errorf("not a  valid syntax for draw")
		}

		if a[0].Type == spacelang.VALUE {
			if a[0].ValueType == spacelang.INT {
				duration = a[0].Value.(int64)
			} else {
				return fmt.Errorf("Expected a integer as a duration")
			}
		} else {
			iface, err := getVariable(vm, a[0].Value.(string))
			if err != nil {
				return err
			}
			if x, ok := iface.(int64); ok {
				duration = x
			} else {
				return fmt.Errorf("Expected a integer as a duration")
			}
		}

		time.Sleep(time.Millisecond * time.Duration(duration))
		return nil
	}

	vm.Funcs["draw"] = func(a ...*spacelang.Token) error {
		var filename string

		if len(a) != 1 {
			return fmt.Errorf("not a  valid syntax for draw")
		}

		if a[0].Type == spacelang.VALUE {
			if a[0].ValueType == spacelang.STRING {
				filename = a[0].Value.(string)
			} else {
				return fmt.Errorf("Expected a string as a filename")
			}
		} else {
			iface, err := getVariable(vm, a[0].Value.(string))
			if err != nil {
				return err
			}
			if x, ok := iface.(string); ok {
				filename = x
			} else {
				return fmt.Errorf("Expected a string as a filename")
			}
		}

		imageFile, err := d.Open(filename)
		if err != nil {
			return err
		}

		m, _, err := image.Decode(imageFile.GetReader())
		if err != nil {
			return err
		}

		offx := (fb.Xres - m.Bounds().Dx()) / 2
		offy := (fb.Yres - m.Bounds().Dy()) / 2

		fb.DrawImage(offx, offy, m)
		return nil
	}
	fb.Init()

	splashFile, err := d.Open("splash.sls")
	if err != nil {
		fmt.Println(err)
		return
	}

	bs, err := ioutil.ReadAll(splashFile.GetReader())
	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		status := uint32(3)
		for status != 0 {
			err := exec.Command("/usr/bin/systemctl", "is-active", "--quiet", "display-manager.service").Run()
			if err == nil {
				status = 0
			}
		}
		os.Exit(0)
	}()

	err = vm.Eval(string(bs))
	if err != nil {
		fmt.Println(err)
		return
	}

	fb.Release()
}
