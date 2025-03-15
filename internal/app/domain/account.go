package domain

import pb "github.com/escape-ship/accountsrv/proto/gen"

type Server struct {
	pb.AccountServer
}

func New() *Server {
	return &Server{}
}

