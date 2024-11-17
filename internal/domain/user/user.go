package user

type UserID uint64

type User struct {
	ID              UserID `db:"id"`
	Firstname       string `db:"first_name"`
	Surname         string `db:"sur_name"`
	Age             uint8  `db:"age"`
	OwnedByUsername string `db:"owned_by_username"`
}

func NewUser(firstname, surname string, age uint8, createdByUsername string) User {
	return User{
		Firstname:       firstname,
		Surname:         surname,
		Age:             age,
		OwnedByUsername: createdByUsername,
	}
}

func (u *User) SetFirstName(firstname string) {
	u.Firstname = firstname
}

func (u *User) SetSurname(surname string) {
	u.Surname = surname
}

func (u *User) SetAge(age uint8) {
	u.Age = age
}

func (u *User) SetOwnByUsername(username string) {
	u.OwnedByUsername = username
}
