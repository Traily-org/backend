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

//go:embed queries/get_appuser.sql
var getAppUserQuery string

//go:embed queries/list_appusers.sql
var listAppUsersQuery string

//go:embed queries/create_appuser.sql
var createAppUserQuery string

//go:embed queries/update_appuser.sql
var updateAppUserQuery string

//go:embed queries/delete_appuser.sql
var deleteAppUserQuery string
