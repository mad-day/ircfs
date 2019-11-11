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

// A PostrgeSQL - backed indexer for IPFS-Files.
// This package implements the Protocol (atop IRC). The Database is
// optional, and only needed, if the Node actually indexes the data.
package indexer

import (
	"database/sql"
	_ "github.com/lib/pq"
	
	"regexp"
	"encoding/json"
	
	irc "github.com/fluffle/goirc/client"
	"github.com/lib/pq/hstore"
	"fmt"
)

var gotya  = regexp.MustCompile(`GOTYA ([^ ]+) (\{.*\}) !`)
var haveu  = regexp.MustCompile(`HAVEU (\{.*\}) \?`)
var report = regexp.MustCompile(`REPORT ([^ ]+) !`)

type TableName string
func (t TableName) Sql() string{
	if len(t)==0 { return "shfidx" }
	return string(t)+"_shfidx"
}

type SearchBase struct{
	*sql.DB
	TableName
}
func (sb *SearchBase) Initialize(connStr string) (err error) {
	sb.DB, err = sql.Open("postgres", connStr)
	if err==nil {
		sb.Exec(`CREATE EXTENSION hstore`)
		sb.Exec(`CREATE TABLE `+sb.Sql()+` (ipfsid text primary key, mtdt hstore, isactive boolean default true)`)
	}
	return
}
func (sb *SearchBase) Gotya(fid string,meta map[string]string) {
	hs := hstore.Hstore{Map:make(map[string]sql.NullString)}
	for k,v := range meta {
		hs.Map[k] = sql.NullString{v,true}
	}
	sb.Exec(`INSERT INTO `+sb.Sql()+` (ipfsid,mtdt,isactive) VALUES ($1,$2::hstore,TRUE)`,fid,hs)
}
func (sb *SearchBase) Haveu(conn *irc.Conn,usr string,meta map[string]string) {
	query := `SELECT ipfsid,hstore_to_json(mtdt) FROM `+sb.Sql()
	args := make([]interface{},0,len(meta)*2)
	i := 0
	for k,v := range meta {
		if i==0 { query += " WHERE " } else { query += " AND " }
		i++
		query += fmt.Sprint("mtdt->$",i)
		i++
		query += fmt.Sprint(" @@ $",i)
		args = append(args,k,v)
	}
	
	ro,err := sb.Query(query,args...)
	if err!=nil { return }
	defer ro.Close()
	var k,v string
	for ro.Next() {
		ro.Scan(&k,&v)
		conn.Privmsg(usr,"IFOUND "+k+" "+v+" !")
	}
}


type Receiver interface{
	Gotya(fid string,meta map[string]string)
}
type Searcher interface{
	Haveu(conn *irc.Conn,usr string,meta map[string]string)
}
type ReportCmd interface{
	Report(fid string)
}

type ChatAgent struct {
	Recvs []Receiver
	Searc []Searcher
	Reprt []ReportCmd
}
func (c *ChatAgent) AddHandlers(conn *irc.Conn) {
	conn.Handle(irc.PRIVMSG,irc.HandlerFunc(c.privmsg))
}
func (c *ChatAgent) AddBackend(be interface{}) {
	if rcv,ok := be.(Receiver); ok {
		c.Recvs = append(c.Recvs,rcv)
	}
	if rcv,ok := be.(Searcher); ok {
		c.Searc = append(c.Searc,rcv)
	}
	if rcv,ok := be.(ReportCmd); ok {
		c.Reprt = append(c.Reprt,rcv)
	}
}

func (c *ChatAgent) privmsg(conn *irc.Conn, line *irc.Line) {
	if len(line.Args)<2 { return }
	var strs []string
	if len(c.Recvs)>0 { strs = gotya.FindStringSubmatch(line.Args[1]) }
	if len(strs)==3 {
		var meta map[string]string
		if err:=json.Unmarshal([]byte(strs[2]),&meta);err==nil {
			for _,r := range c.Recvs {
				go r.Gotya(strs[1],meta)
			}
			return
		}// else {
		//	fmt.Println("json",err)
		//}
	}
	strs = nil
	if len(c.Searc)>0 { strs = haveu.FindStringSubmatch(line.Args[1]) }
	if len(strs)==2 {
		var meta map[string]string
		if json.Unmarshal([]byte(strs[1]),&meta)==nil {
			if len(meta)==0 { return }
			for _,r := range c.Searc {
				go r.Haveu(conn,line.Nick,meta)
			}
			return
		}
	}
	strs = nil
	if len(c.Reprt)>0 { strs = report.FindStringSubmatch(line.Args[1]) }
	if len(strs)==2 {
		// Safety: only messages, directed to a channel are allowed for report messages.
		if line.Args[0][0]!='#' { return }
		for _,r := range c.Reprt {
			go r.Report(strs[1])
		}
		return
	}
	//fmt.Println(line.Cmd,line.Args,"(",line.Nick,line.Ident,line.Host,")")
}

