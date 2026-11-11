package common

type Permission struct {
	ServerMaxMem      int64 // B
	MaxServerCount    int64
	MaxOnlineCount    int64
	MaxModsCount      int64
	CanAddUser        bool
	CanCreateServer   bool
	CanEditServerArgs bool
	CanBackUpServer   bool
	MaxBackUpCount    int64
}

var (
	B  int64 = 1
	KB       = B * 1024
	MB       = KB * 1024
	GB       = MB * 1024
)

var memberPermission = Permission{
	ServerMaxMem:      3 * GB,
	MaxServerCount:    5,
	MaxOnlineCount:    1,
	MaxModsCount:      50,
	CanAddUser:        false,
	CanCreateServer:   true,
	CanEditServerArgs: false,
	CanBackUpServer:   false,
	MaxBackUpCount:    0,
}

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
	RoleMemberUser:         memberPermission,
	RoleExternalMemberUser: memberPermission,
	RoleVipMemberUser: Permission{
		ServerMaxMem:      8 * GB,
		MaxServerCount:    10,
		MaxOnlineCount:    2,
		MaxModsCount:      256,
		CanAddUser:        false,
		CanCreateServer:   true,
		CanEditServerArgs: false,
		CanBackUpServer:   true,
		MaxBackUpCount:    5,
	},
}

func IsRoot(role int) bool {
	return role == RoleRootUser
}

func PermFor(role int) (Permission, bool) {
	perm, ok := RolePerm[role]
	return perm, ok
}

func IsAssignableRole(role int) bool {
	if role == RoleExternalMemberUser {
		return false
	}
	if IsRoot(role) {
		return true
	}
	_, ok := PermFor(role)
	return ok
}
