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

// Client Package.
package client

import (
	"regexp"
	"encoding/json"
	
	irc "github.com/fluffle/goirc/client"
	"fmt"
)

var ifound  = regexp.MustCompile(`IFOUND ([^ ]+) (\{.*\}) !`)

type Finder interface{
	Ifound(fid string,meta map[string]string)
}

type ChatAgent struct {
	Finds []Finder
}
func (c *ChatAgent) AddHandlers(conn *irc.Conn) {
	conn.Handle(irc.PRIVMSG,irc.HandlerFunc(c.privmsg))
}
func (c *ChatAgent) AddBackend(be interface{}) {
	if rcv,ok := be.(Finder); ok {
		c.Finds = append(c.Finds,rcv)
	}
}

func (c *ChatAgent) privmsg(conn *irc.Conn, line *irc.Line) {
	if len(line.Args)<2 { return }
	var strs []string
	if len(c.Finds)>0 { strs = ifound.FindStringSubmatch(line.Args[1]) }
	if len(strs)==3 {
		var meta map[string]string
		if err:=json.Unmarshal([]byte(strs[2]),&meta);err==nil {
			for _,r := range c.Finds {
				go r.Ifound(strs[1],meta)
			}
			return
		}// else {
		//	fmt.Println("json",err)
		//}
	}
}

func IGotYouAFile(conn *irc.Conn,chnl string, fid string ,meta map[string]string) {
	data,_ := json.Marshal(meta)
	conn.Privmsg(chnl,fmt.Sprintf("GOTYA %s %s !",fid,data))
}
func HaveYou(conn *irc.Conn,chnl string,meta map[string]string) {
	data,_ := json.Marshal(meta)
	conn.Privmsg(chnl,fmt.Sprintf("HAVEU %s ?",data))
}
func ReportMissing(conn *irc.Conn,chnl string,fid string) {
	conn.Privmsg(chnl,fmt.Sprintf("REPORT %s !",fid))
}

