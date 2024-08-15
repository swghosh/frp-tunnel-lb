package controller

import "os"

var (
	RemoteServerHostName = os.Getenv("FRPS_SERVER_HOST")
	RemoteServerAuthKey  = os.Getenv("FRPS_SERVER_AUTH_KEY")
	RemoteServerPort     = os.Getenv("FRPS_SERVER_PORT")

	FRPCContainerImage = os.Getenv("FRP_IMAGE")
	FRPExposedHost     = os.Getenv("FRP_EXPOSED_HOST")
)
