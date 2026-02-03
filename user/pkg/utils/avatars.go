package utils

import "math/rand"

var AvailableAvatars = []string{
	"avatar-blue.png",
	"avatar-red.png",
	"avatar-green.png",
	"avatar-yellow.png",
	"avatar-orange.png",
	"avatar-pink.png",
	"avatar-turquoise.png",
	"avatar-white.png",
	"avatar-violet.png",
}

func RandomAvatar() string {
	return AvailableAvatars[rand.Intn(len(AvailableAvatars))]
}
