package rbac

// Complete catalog of the 34 system permissions defined in FaceClock Fase 1 (§ 2.3).
const (
	// User permissions
	PermUserRead          = "user.read"
	PermUserCreate        = "user.create"
	PermUserUpdate        = "user.update"
	PermUserDelete        = "user.delete"
	PermUserAssignRole    = "user.assign_role"
	PermUserResetPassword = "user.reset_password"

	// Employee permissions
	PermEmployeeRead     = "employee.read"
	PermEmployeeReadSelf = "employee.read_self"
	PermEmployeeCreate   = "employee.create"
	PermEmployeeUpdate   = "employee.update"
	PermEmployeeDelete   = "employee.delete"

	// Role permissions
	PermRoleRead             = "role.read"
	PermRoleCreate           = "role.create"
	PermRoleUpdate           = "role.update"
	PermRoleDelete           = "role.delete"
	PermRoleAssignPermission = "role.assign_permission"

	// Permission catalog
	PermPermissionRead = "permission.read"

	// Settings & Audit permissions
	PermSettingsRead   = "settings.read"
	PermSettingsUpdate = "settings.update"
	PermAuditRead      = "audit.read"

	// Face / Biometric permissions
	PermFaceEnrollSelf = "face.enroll_self"
	PermFaceReadSelf   = "face.read_self"
	PermFaceEnrollAny  = "face.enroll_any"
	PermFaceReadAny    = "face.read_any"
	PermFaceDeleteAny  = "face.delete_any"
	PermFaceReindex    = "face.reindex"

	// Attendance permissions
	PermAttendanceCheckin  = "attendance.checkin"
	PermAttendanceReadSelf = "attendance.read_self"
	PermAttendanceReadAll  = "attendance.read_all"
	PermAttendanceApprove  = "attendance.approve"
	PermAttendanceExport   = "attendance.export"

	// Location permissions
	PermLocationRead   = "location.read"
	PermLocationCreate = "location.create"
	PermLocationUpdate = "location.update"
	PermLocationDelete = "location.delete"
)

type PermissionDef struct {
	Name        string
	Description string
}

// SystemPermissions contains all 34 permissions with their human-readable descriptions.
var SystemPermissions = []PermissionDef{
	{PermUserRead, "Lihat daftar & detail user"},
	{PermUserCreate, "Buat user"},
	{PermUserUpdate, "Ubah user (termasuk aktif/nonaktif)"},
	{PermUserDelete, "Soft-delete user"},
	{PermUserAssignRole, "Set role milik user"},
	{PermUserResetPassword, "Reset password user lain"},

	{PermEmployeeRead, "Lihat semua karyawan"},
	{PermEmployeeReadSelf, "Lihat data karyawan diri sendiri"},
	{PermEmployeeCreate, "Tambah karyawan"},
	{PermEmployeeUpdate, "Ubah karyawan"},
	{PermEmployeeDelete, "Soft-delete karyawan"},

	{PermRoleRead, "Lihat role"},
	{PermRoleCreate, "Buat role"},
	{PermRoleUpdate, "Ubah role"},
	{PermRoleDelete, "Hapus role"},
	{PermRoleAssignPermission, "Set permission milik role"},

	{PermPermissionRead, "Lihat katalog permission"},

	{PermSettingsRead, "Baca app settings"},
	{PermSettingsUpdate, "Ubah app settings"},
	{PermAuditRead, "Baca audit log"},

	{PermFaceEnrollSelf, "Enroll wajah sendiri"},
	{PermFaceReadSelf, "Lihat referensi wajah sendiri"},
	{PermFaceEnrollAny, "Enroll wajah karyawan lain"},
	{PermFaceReadAny, "Lihat referensi wajah siapa pun"},
	{PermFaceDeleteAny, "Nonaktifkan referensi wajah siapa pun"},
	{PermFaceReindex, "Menjalankan job regenerasi embedding saat model berganti"},

	{PermAttendanceCheckin, "Melakukan check-in/out untuk diri sendiri"},
	{PermAttendanceReadSelf, "Lihat riwayat absensi sendiri"},
	{PermAttendanceReadAll, "Lihat absensi semua karyawan"},
	{PermAttendanceApprove, "Approve/reject absensi pending_review"},
	{PermAttendanceExport, "Export rekap"},

	{PermLocationRead, "Kelola office_locations"},
	{PermLocationCreate, "Kelola office_locations"},
	{PermLocationUpdate, "Kelola office_locations"},
	{PermLocationDelete, "Kelola office_locations"},
}

// AllPermissions returns the slice of all 34 valid permission names.
var AllPermissions = func() []string {
	perms := make([]string, len(SystemPermissions))
	for i, def := range SystemPermissions {
		perms[i] = def.Name
	}
	return perms
}()

// IsValidPermission checks whether a given permission name is in the catalog.
func IsValidPermission(p string) bool {
	for _, perm := range AllPermissions {
		if perm == p {
			return true
		}
	}
	return false
}
