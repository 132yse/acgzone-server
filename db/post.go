package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"github.com/cliclitv/go-clicli/def"
	"github.com/wangbin/jiebago"
)

var seg jiebago.Segmenter

func init() {
	seg.LoadDictionary("dict.txt")
}

func AddPost(title string, content string, status string, sort string, tag string, uid int) (*def.Post, error) {
	cstZone := time.FixedZone("CST", 8*3600)  
	ctime := time.Now().In(cstZone).Format("2006-01-02 15:04")
	stmtIns, err := dbConn.Prepare("INSERT INTO posts (title,content,status,sort,tag,time,uid) VALUES (?,?,?,?,?,?,?)")
	if err != nil {
		return nil, err

	}
	_, err = stmtIns.Exec(title, content, status, sort, tag, ctime, uid)
	if err != nil {
		return nil, err
	}
	res := &def.Post{Title: title, Content: content, Status: status, Sort: sort, Tag: tag, Time: ctime, Uid: uid}
	defer stmtIns.Close()

	return res, err
}

func UpdatePost(id int, title string, content string, status string, sort string, tag string, time string) (*def.Post, error) {
	stmtIns, err := dbConn.Prepare("UPDATE posts SET title=?,content=?,status=?,sort=?,tag=?,time=? WHERE id =?")
	if err != nil {
		return nil, err

	}
	_, err = stmtIns.Exec(&title, &content, &status, &sort, &tag, &time, &id)
	if err != nil {
		return nil, err
	}
	res := &def.Post{Id: id, Title: title, Content: content, Status: status, Sort: sort, Tag: tag, Time: time}
	defer stmtIns.Close()
	return res, err
}

func DeletePost(id int) error {
	stmtDel, err := dbConn.Prepare("DELETE FROM posts WHERE id=?")
	if err != nil {
		return err
	}

	_, err = stmtDel.Exec(id)
	if err != nil {
		return err
	}
	stmtDel.Close()

	return nil
}

func GetPost(id int) (*def.Post, error) {
	stmtOut, err := dbConn.Prepare(`SELECT posts.id,posts.title,posts.content,posts.status,posts.sort,posts.tag,posts.time,users.id,users.name,users.qq FROM posts 
INNER JOIN users ON posts.uid = users.id WHERE posts.id = ?`)
	if err != nil {
		return nil, err
	}
	var pid, uid int
	var title, content, status, sort, tag, ctime, uname, uqq string

	err = stmtOut.QueryRow(id).Scan(&pid, &title, &content, &status, &sort, &tag, &ctime, &uid, &uname, &uqq)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	defer stmtOut.Close()

	res := &def.Post{Id: pid, Title: title, Content: content, Status: status, Sort: sort, Tag: tag, Time: ctime, Uid: uid, Uname: uname, Uqq: uqq}

	return res, nil
}

func GetPosts(page int, pageSize int, status string, sort string, tag string, uid int) ([]*def.Post, error) {
	start := pageSize * (page - 1)
	tags := strings.Fields(tag)

	var query string
	if status != "" && status != "nowait" && len(status) < 10 {
		query = fmt.Sprintf(`AND posts.status ='%s'`, status)
	}

	if sort != "" && sort != "bgm" && len(sort) < 10 {
		query += fmt.Sprintf(`AND posts.sort ='%s'`, sort)
	}

	if uid != 0 && len(string(uid)) < 6 {
		query += fmt.Sprintf(`AND posts.uid ='%d'`, uid)
	}

	if sort == "bgm" {
		query += `AND NOT posts.sort='原创'`
	}
	if status == "nowait" {
		query += `AND NOT posts.status='wait'`
	}

	if len(tags) != 0 {
		query += `AND (1=2 `
		for i := 0; i < len(tags); i++ {
			if len(tags[i]) > 10 {
				continue
			}
			key := string("%" + tags[i] + "%")
			query += fmt.Sprintf(`OR posts.tag LIKE '%s'`, key)
		}
		query += `)`
	}

	sqlRaw := fmt.Sprintf(`SELECT posts.id,posts.title,posts.content,posts.status,posts.sort,posts.tag,posts.time,users.id,users.name,users.qq FROM posts LEFT JOIN users ON posts.uid = users.id 
WHERE 1=1 %s ORDER BY time DESC limit ?,?`, query)

	//fmt.Printf("%s", sqlRaw)

	stmt, _ := dbConn.Prepare(sqlRaw)

	var rows, _ = stmt.Query(start, pageSize)

	defer stmt.Close()

	var res []*def.Post

	for rows.Next() {
		var id, uid int
		var title, content, status, sort, tag, ctime, uname, uqq string
		if err := rows.Scan(&id, &title, &content, &status, &sort, &tag, &ctime, &uid, &uname, &uqq); err != nil {
			return res, err
		}
		c := &def.Post{Id: id, Title: title, Content: content, Status: status, Sort: sort, Tag: tag, Time: ctime, Uid: uid, Uname: uname, Uqq: uqq}
		res = append(res, c)
	}

	return res, nil

}

func SearchPosts(key string) ([]*def.Post, error) {
	var likeStr string

	var words <-chan string
	words = seg.CutForSearch(key, true)

	index := 0
	for word := range words {

		index++
		if index == 1 {
			likeStr = "title LIKE '%" + word + "%'"
		} else {
			likeStr += " OR title LIKE '%" + word + "%'"
		}

	}

	sql := `SELECT posts.id, posts.title, posts.content, posts.status, posts.sort, posts.tag, posts.time, users.id, users.name, users.qq 
					FROM posts 
					LEFT JOIN users ON posts.uid = users.id 
					LEFT JOIN pv ON pv.pid = posts.id
					WHERE ` + likeStr + `
					ORDER BY pv.pv DESC`

	stmtOut, err := dbConn.Prepare(sql)

	var res []*def.Post

	rows, err := stmtOut.Query()
	if err != nil {
		return res, err
	}

	defer stmtOut.Close()

	for rows.Next() {
		var id, uid int
		var title, content, status, sort, tag, ctime, uname, uqq string
		if err := rows.Scan(&id, &title, &content, &status, &sort, &tag, &ctime, &uid, &uname, &uqq); err != nil {
			return res, err
		}
		c := &def.Post{Id: id, Title: title, Content: content, Status: status, Sort: sort, Tag: tag, Time: ctime, Uid: uid, Uname: uname, Uqq: uqq}
		res = append(res, c)
	}

	return res, nil
}
func GetRank() ([]*def.Post, error) {
	stmtOut, err := dbConn.Prepare("SELECT posts.id, posts.title, posts.content, posts.status, posts.sort, posts.tag,posts.time FROM posts LEFT JOIN pv ON posts.id = pv.pid ORDER BY pv DESC limit 0,10")

	var res []*def.Post

	rows, err := stmtOut.Query()
	if err != nil {
		return res, err
	}

	defer stmtOut.Close()

	for rows.Next() {
		var id int
		var title, content, status, sort, tag, ctime string
		if err := rows.Scan(&id, &title, &content, &status, &sort, &tag, &ctime); err != nil {
			return res, err
		}

		c := &def.Post{Id: id, Title: title, Content: content, Status: status, Sort: sort, Tag: tag, Time: ctime}
		res = append(res, c)
	}

	return res, nil
}
