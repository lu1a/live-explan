package globals

var (
	Version     = DevelopmentVersion
	VersionTime = ""
)

// DevelopmentVersion is a version that is the default when running the application with go run or docker compose.
const DevelopmentVersion = "development"

