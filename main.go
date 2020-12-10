// wpaste - easy code sharing
// Copyright (C) 2020  Evgeniy Rybin
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString creates a random string with the charset
// that contains all letters and digits
func RandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// OpenRandomNameFile creates a file in the directory dir,
// opens the file for reading and writing, and returns the resulting *os.File.
// The filename is generated by taking random string with length nameLen.
// Multiple programs calling OpenRandomNameFile simultaneously
// will not choose the same file. The caller can use f.Name()
// to find the pathname of the file.
func OpenRandomNameFile(dir string, nameLen int) (f *os.File, err error) {
	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(dir, RandomString(nameLen))
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				rand.Seed(time.Now().UTC().UnixNano())
			}
			continue
		}
		break
	}
	return
}

// Help redirect to github
func Help(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://github.com/waika28/wpaste.cyou", http.StatusSeeOther)
}

// UploadFile save file and response it ID
func UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > 10<<20 {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write([]byte("413 - Max content size is 10MiB"))
		return
	}

	data := r.FormValue("f")
	file := bytes.NewReader([]byte(data))

	name := r.FormValue("name")

	var (
		servFile *os.File
		err      error
	)
	if len(name) == 0 {
		servFile, err = OpenRandomNameFile(FilesDir(), 3)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Something bad happened!"))
		}
	} else {
		name = filepath.Join(FilesDir(), RandomString(3))
		servFile, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("409 - This filename already taken!"))
		}
	}
	defer servFile.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Something bad happened!"))
	}

	servFile.Write(fileBytes)

	if len(name) == 0 {
		name = filepath.Base(servFile.Name())
	}

	w.Write([]byte(name))
}

// SendFile respond file by it ID
func SendFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	file, err := ioutil.ReadFile(FilesDir() + "/" + vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404 - File %v not found", vars["id"])
		return
	}
	w.Write(file)
}

// WpasteRouter make router with all needed Handlers
func WpasteRouter() *mux.Router {
	Router := mux.NewRouter().StrictSlash(true)

	Router.HandleFunc("/", Help).Methods("GET")
	Router.HandleFunc("/", UploadFile).Methods("POST")

	Router.HandleFunc("/{id}", SendFile)
	return Router
}

// Basedir return root working directory
func Basedir() string {
	basedir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return basedir
}

// FilesDir return directory where uploaded files stored
func FilesDir() string {
	return Basedir() + "/files"
}

// Install prepare to start:
// 1. Make directory for user files
// 2. Set random seed
func Install() {
	os.Mkdir(FilesDir(), 0766)
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	Install()
	http.ListenAndServe(":9990", WpasteRouter())
}
