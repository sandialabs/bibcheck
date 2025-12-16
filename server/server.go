// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	e            *echo.Echo
	shirtyApiKey string
}

func NewServer(shirtyApiKey string) *Server {

	e := echo.New()

	s := &Server{
		e:            e,
		shirtyApiKey: shirtyApiKey,
	}

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Static files
	e.Static("/static", "server/static")
	e.Static("/uploads", "uploads") // TODO: configurable (also in upload handler)

	// Routes
	e.GET("/", handleUploadGet)
	e.POST("/upload", s.handleUploadPost)
	e.GET("/analyze/:id", handleAnalyzeGet)
	e.GET("/api/status/:id", handleGetAnalysisStatus)
	e.GET("/api/entries/:id", handleGetBibEntries)

	return s
}

func (s *Server) Run() error {
	return s.e.Start(":8080")
}
