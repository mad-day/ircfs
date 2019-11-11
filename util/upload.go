/*
This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

In jurisdictions that recognize copyright laws, the author or authors
of this software dedicate any and all copyright interest in the
software to the public domain. We make this dedication for the benefit
of the public at large and to the detriment of our heirs and
successors. We intend this dedication to be an overt act of
relinquishment in perpetuity of all present and future rights to this
software under copyright law.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.

For more information, please refer to <https://unlicense.org>
*/

// Upload Helper.
package util

import (
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/dhowden/tag"
	
	"path/filepath"
	"strings"
	"regexp"
	"fmt"
	"os"
)

var whitesp = regexp.MustCompile(`[^a-zA-Z0-9]+`)
var splitr = regexp.MustCompile(`(.*)\.([^\.]+)$`)

type Util struct {
	Ipfs *shell.Shell
}

func (u *Util) InsertFile(file string,pmeta *map[string]string) (string,error){
	fobj,err := os.Open(file)
	if err!=nil { return "",err }
	defer fobj.Close()
	var title,ext string
	{
		_,ft := filepath.Split(file)
		sm := splitr.FindStringSubmatch(ft)
		if len(sm)==2 { title,ext = sm[1],strings.ToLower(sm[2]) } else { title = ft }
	}
	title = whitesp.ReplaceAllString(title," ")
	if md,err := tag.ReadFrom(fobj); err==nil {
		meta := make(map[string]string)
		if t := md.Title(); t!="" { meta["title"] = t } else { meta["title"] = title }
		if t := md.Album(); t!="" { meta["album"] = t }
		if t := md.Artist(); t!="" { meta["artist"] = t }
		if t := md.AlbumArtist(); t!="" { meta["albumartist"] = t }
		if t := md.Album(); t!="" { meta["album"] = t }
		if t := md.Composer(); t!="" { meta["composer"] = t }
		if t := md.Year(); t!=0 { meta["year"] = fmt.Sprint(t) }
		if t := md.Genre(); t!="" { meta["genre"] = t }
		if t := string(md.FileType()); t!="" && ext=="" { ext = strings.ToLower(t) }
		meta["type"] = "audio "+ext
		meta["ext"] = ext
		*pmeta = meta
	} else {
		var foom string
		switch strings.ToLower(ext) {
		case "mp3","m4a","alac","flac","ogg","opus": foom = "audio "
		}
		*pmeta = map[string]string{"title":title,"type":foom+ext,"ext":ext}
	}
	_,err = fobj.Seek(0,0)
	if err!=nil { return "",err }
	return u.Ipfs.Add(fobj)
}


