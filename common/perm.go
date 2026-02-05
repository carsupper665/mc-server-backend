package common

// TODO 實裝這些功能
type Permission struct {
	ServerMaxMem      int64 // B
	MaxServerCount    int8
	MaxOnlineCount    int8
	MaxModsCount      int16
	CanAddUser        bool
	CanCreateServer   bool
	CanEditServerArgs bool
	CanBackUpServer   bool
	MaxBackUpCount    int8
}

var (
	B  int64 = 1
	KB       = B * 1024
	MB       = KB * 1024
	GB       = MB * 1024
)
var RolePerm = map[int]Permission{
	RoleGuestUser: Permission{
		ServerMaxMem:      512 * MB,
		MaxServerCount:    1,
		MaxOnlineCount:    1,
		MaxModsCount:      10,
		CanAddUser:        false,
		CanCreateServer:   true,
		CanEditServerArgs: false,
		CanBackUpServer:   false,
		MaxBackUpCount:    0,
	},
	RoleMemberUser: Permission{
		ServerMaxMem:      3 * GB,
		MaxServerCount:    5,
		MaxOnlineCount:    2,
		MaxModsCount:      50,
		CanAddUser:        false,
		CanCreateServer:   true,
		CanEditServerArgs: false,
		CanBackUpServer:   false,
		MaxBackUpCount:    0,
	},
	RoleVipMemberUser: Permission{
		ServerMaxMem:      8 * GB,
		MaxServerCount:    10,
		MaxOnlineCount:    5,
		MaxModsCount:      256,
		CanAddUser:        false,
		CanCreateServer:   true,
		CanEditServerArgs: false,
		CanBackUpServer:   true,
		MaxBackUpCount:    5,
	},
}
