package value_objects

type PasswordHash string

func (h PasswordHash) String() string { return string(h) }
