package modules

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/icetrays/icetrays/consensus"
	"github.com/icetrays/icetrays/types"
	"github.com/ipfs/go-path"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Op struct {
	Op     string   `json:"op"`
	Params []string `json:"params"`
}

func Server2(node *consensus.Node, config Config) error {
	router := gin.Default()
	mulAddr, err := multiaddr.NewMultiaddr(config.Ipfs)
	if err != nil {
		return err
	}
	netAddr, err := manet.ToNetAddr(mulAddr)
	if err != nil {
		return err
	}
	ipfsUrl, err := url.Parse(fmt.Sprintf("http://%s", netAddr.String()))
	if err != nil {
		return err
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(ipfsUrl)
	reverseProxy.Transport = http.DefaultTransport

	router.POST("/itscp", func(c *gin.Context) {
		params := &types.RequestCpParams{}
		err := c.MustBindWith(params, binding.JSON)
		if err != nil {
			return
		}
		err = node.Cp(c, params.File, path.Path(params.Dir), types.PinInfo{
			Cid:      params.File,
			PinCount: uint32(params.PinCount),
			Crust:    params.Crust,
		})
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(200, "")
	})
	router.POST("/itsls", func(c *gin.Context) {
		params := &types.RequestLsParams{}
		err := c.MustBindWith(params, binding.JSON)
		if err != nil {
			return
		}
		files, err := node.Ls(c, path.Path(params.Dir))
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(200, files)
	})
	router.POST("/itsmv", func(c *gin.Context) {
		params := &types.RequestMvParams{}
		err := c.MustBindWith(params, binding.JSON)
		if err != nil {
			return
		}
		err = node.Mv(c, path.Path(params.Src), path.Path(params.Dst))
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(200, "")
	})
	router.POST("/itsrm", func(c *gin.Context) {
		params := &types.RequestRmParams{}
		err := c.MustBindWith(params, binding.JSON)
		if err != nil {
			return
		}
		err = node.Rm(c, path.Path(params.Dir))
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(200, "")
	})

	router.POST("/itsmkdir", func(c *gin.Context) {
		params := &types.RequestRmParams{}
		err := c.MustBindWith(params, binding.JSON)
		if err != nil {
			return
		}
		err = node.Mkdir(c, path.Path(params.Dir))
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(200, "")
	})

	router.POST("/itspin", func(c *gin.Context) {
		params := &types.RequestPinParams{}
		err := c.MustBindWith(params, binding.JSON)
		if err != nil {
			return
		}
		err = node.Pin(c, types.PinInfo{
			Cid:      params.File,
			PinCount: uint32(params.PinCount),
			Crust:    params.Crust,
		})
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(200, "")
	})

	router.POST("/itsunpin", func(c *gin.Context) {
		params := &types.RequestUnPinParams{}
		err := c.MustBindWith(params, binding.JSON)
		if err != nil {
			return
		}
		err = node.UnPin(c, params.File)
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.JSON(200, "")
	})

	var proxyHandle = func(c *gin.Context) {
		reverseProxy.ServeHTTP(c.Writer, c.Request)
	}

	router.NoRoute(proxyHandle)
	go router.Run(fmt.Sprintf(":%d", config.Port))
	return nil
}
