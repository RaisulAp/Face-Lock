package auth

import "strings"

// weakPasswords contains top commonly used passwords that must be rejected.
var weakPasswords = map[string]struct{}{
	"123456": {}, "password": {}, "12345678": {}, "qwerty": {}, "123456789": {},
	"12345": {}, "1234": {}, "111111": {}, "1234567": {}, "dragon": {},
	"welcome": {}, "admin": {}, "sunshine": {}, "princess": {}, "football": {},
	"master": {}, "monkey": {}, "666666": {}, "shadow": {}, "baseball": {},
	"charlie": {}, "superman": {}, "michael": {}, "jennifer": {}, "secret": {},
	"jessica": {}, "mustang": {}, "bullet": {}, "killer": {}, "system": {},
	"jordan": {}, "daniel": {}, "access": {}, "thomas": {}, "orange": {},
	"hunter": {}, "starwars": {}, "cheese": {}, "flower": {}, "buster": {},
	"ginger": {}, "summer": {}, "matrix": {}, "robert": {}, "harley": {},
	"mariah": {}, "guitar": {}, "batman": {}, "coffee": {}, "silver": {},
	"spider": {}, "computer": {}, "phoenix": {}, "monkey1": {}, "diamond": {},
	"jackson": {}, "anthony": {}, "austin": {}, "purple": {}, "bubbles": {},
	"falcon": {}, "cougar": {}, "merlin": {}, "yellow": {}, "morgan": {},
	"simpson": {}, "tiger": {}, "george": {}, "peanut": {}, "pepper": {},
	"chester": {}, "whiskey": {}, "cookie": {}, "cannon": {}, "turtle": {},
	"ranger": {}, "thunder": {}, "panther": {}, "phantom": {}, "blazer": {},
	"snoopy": {}, "beast": {}, "boston": {}, "bullet1": {}, "camaro": {},
	"copper": {}, "corner": {}, "cotton": {}, "cowboy": {}, "crystal": {},
	"dakota": {}, "diesel": {}, "donald": {}, "driver": {}, "falcon1": {},
	"ferrari": {}, "french": {}, "gateway": {}, "golden": {}, "hammer": {},
	"hannah": {}, "hunter1": {}, "jaguar": {}, "jasmine": {}, "jasper": {},
	"knight": {}, "legend": {}, "lightning": {}, "lincoln": {}, "london": {},
	"magic": {}, "magnum": {}, "marine": {}, "maxwell": {}, "member": {},
	"metallica": {}, "midnight": {}, "mickey": {}, "miller": {}, "monster": {},
	"murphy": {}, "network": {}, "newyork": {}, "oliver": {}, "olympic": {},
	"parker": {}, "pass123": {}, "patriot": {}, "player": {}, "polaris": {},
	"potato": {}, "rachel": {}, "raiders": {}, "rainbow": {}, "rescue": {},
	"river": {}, "rocket": {}, "safety": {}, "sample": {}, "samuel": {},
	"scorpio": {}, "serenity": {}, "shadow1": {}, "simon": {}, "simple": {},
	"soccer": {}, "soldier": {}, "source": {}, "spirit": {}, "spring": {},
	"standard": {}, "stranger": {}, "stream": {}, "success": {}, "sunset": {},
	"target": {}, "taurus": {}, "timber": {}, "titanic": {}, "toronto": {},
	"toyota": {}, "trinity": {}, "trojan": {}, "trustee": {}, "united": {},
	"vampire": {}, "vanilla": {}, "vector": {}, "version": {}, "victor": {},
	"violet": {}, "walnut": {}, "warrior": {}, "webster": {}, "western": {},
	"winter": {}, "wizard": {}, "wonder": {}, "yellow1": {}, "zebra": {},
	"password1": {}, "password12": {}, "password123": {}, "Password123": {},
	"admin123": {}, "welcome1": {}, "welcome123": {}, "changeme": {}, "changeme123": {},
	"faceclock": {}, "faceclock123": {}, "Faceclock123": {}, "root1234": {},
}

// IsWeakPassword returns true if the password is in the static dictionary of weak passwords.
func IsWeakPassword(p string) bool {
	lower := strings.ToLower(strings.TrimSpace(p))
	_, found := weakPasswords[lower]
	return found
}
