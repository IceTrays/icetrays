package modules

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/icetrays/icetrays/consensus"
	"github.com/icetrays/icetrays/consensus/pb"
	"strconv"
	"strings"
)

type Op struct {
	Op     string   `json:"op"`
	Params []string `json:"params"`
	Root   string   `json:"root"`
}

type UploadParam struct {
	FileSize int64 `param:"fileSize"`
}

func Server2(node *consensus.Node, config Config) {
	router := gin.Default()

	// Query string parameters are parsed using the existing underlying request object.
	// The request responds to a url matching:  /welcome?firstname=Jane&lastname=Doe
	router.POST("/fs", func(c *gin.Context) {
		d, err := c.GetRawData()
		if err != nil {
			return
		}
		op := &Op{}
		err = json.Unmarshal(d, op)
		if err != nil {
			c.JSON(200, err.Error())
		}
		switch op.Op {
		case "ls":
			n, err := node.Ls(c, op.Params[0])
			if err != nil {
				fmt.Printf("%+v", err)
			}
			c.JSON(200, n)
		case "cp":
			err := node.Op(c, pb.Instruction_CP, op.Params[0], op.Params[1])
			if err != nil {
				fmt.Printf("%+v", err)
				c.JSON(200, err.Error())
			} else {
				c.JSON(200, "success")
			}
		case "mv":
			err := node.Op(c, pb.Instruction_MV, op.Params[0], op.Params[1])
			if err != nil {
				c.JSON(200, err.Error())
			} else {
				c.JSON(200, "success")
			}
		case "rm":
			err := node.Op(c, pb.Instruction_RM, op.Params[0])
			if err != nil {
				c.JSON(200, err.Error())
			} else {
				c.JSON(200, "success")
			}
		case "mkdir":
			err := node.Op(c, pb.Instruction_MKDIR, op.Params[0])
			if err != nil {
				c.JSON(200, err.Error())
			} else {
				c.JSON(200, "success")
			}
		default:
			c.JSON(200, "???")
		}
	})

	router.POST("/pin", func(c *gin.Context) {
		ss := strings.Split(c.Request.Header.Get("Content-Type"), "boundary=")
		if len(ss) < 2 {
			c.JSON(500, errors.New("gg"))
			return
		}
		fileName := c.Query("fileName")
		fileSize := c.Query("fileSize")
		pinCount := c.Query("pinCount")
		r, err := strconv.ParseInt(fileSize, 10, 64)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		fr, err := NewFormReader(c.Request.Body, ss[1], r)

		r2, err := strconv.ParseInt(pinCount, 10, 64)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		err = node.UploadFile(c, fileName, int(r2), fr)
		if err != nil {
			fmt.Println(err.Error())
			c.JSON(500, err.Error())
		} else {
			c.JSON(200, "success")
		}
	})

	go router.Run(fmt.Sprintf(":%d", config.Port))
}
