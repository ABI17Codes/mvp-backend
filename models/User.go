package models

const (
	RoleAdmin      = "admin"
	RoleStoreOwner = "storeowner"
	RoleCustomer   = "customer"
	RoleStaff      = "staff" // or whatever your 4th role is
)

type User struct {
	BaseModel
	Name     string `json:"name"`
	Email    string `json:"email" gorm:"unique"`
	Password string `json:"-"`
	Role     string `json:"role" gorm:"default:customer"`
}