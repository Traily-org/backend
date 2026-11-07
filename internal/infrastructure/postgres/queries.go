package postgres

import _ "embed"

//go:embed queries/get_user.sql
var getUserQuery string

//go:embed queries/create_user.sql
var createUserQuery string

//go:embed queries/update_user.sql
var updateUserQuery string

//go:embed queries/delete_user.sql
var deleteUserQuery string

//go:embed queries/get_trail.sql
var getTrailQuery string

//go:embed queries/list_trails_by_user.sql
var listTrailsByUserQuery string

//go:embed queries/create_trail.sql
var createTrailQuery string
