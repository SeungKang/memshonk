package app

import (
	"crypto/rand"
	"encoding/binary"
	mathrand "math/rand"
)

func newRandomStringer() *randomStringer {
	b := make([]byte, 8)

	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	seed := binary.BigEndian.Uint64(b)

	src := mathrand.New(mathrand.NewSource(int64(seed)))

	return &randomStringer{
		rand:       src,
		adjectives: adjectives(),
		animals:    animals(),
	}
}

type randomStringer struct {
	rand       *mathrand.Rand
	adjectives []string
	animals    []string
}

func (o *randomStringer) String() string {
	adjective := o.adjectives[o.rand.Intn(len(o.adjectives))]

	animal := o.animals[o.rand.Intn(len(o.animals))]

	return adjective + "-" + animal
}

func adjectives() []string {
	return []string{
		"big", "small", "happy", "sad", "tall", "short", "old", "new", "fast", "slow",
		"colorful", "bright", "dark", "light", "loud", "quiet", "strong", "weak",
		"amazing", "awesome", "beautiful", "bizarre", "calm", "crazy", "curious", "delightful",
		"elegant", "exotic", "fantastic", "fascinating", "fierce", "friendly", "funny", "gorgeous",
		"harmless", "intelligent", "intriguing", "kind", "lively", "lovely", "magnificent", "mysterious",
		"playful", "quaint", "quirky", "remarkable", "sophisticated", "spectacular", "spunky", "stunning",
		"unusual", "vibrant", "wonderful", "x-traordinary", "youthful", "zany",
	}
}

func animals() []string {
	return []string{
		"lion", "tiger", "bear", "monkey", "dog", "cat", "elephant", "giraffe", "zebra", "kangaroo",
		"penguin", "koala", "crocodile", "snake", "bird", "fish", "horse", "cow", "pig", "sheep",
		"rabbit", "tortoise", "frog", "butterfly", "bee", "ant", "grasshopper", "ladybug", "firefly",
		"wolf", "fox", "deer", "squirrel", "chipmunk", "otter", "seal", "dolphin", "whale", "shark",
		"octopus", "squid", "crab", "lobster", "turtle", "alligator", "rhinoceros", "hippopotamus", "hyena",
		"cheetah", "leopard", "jaguar", "panda", "gorilla", "chimpanzee", "orangutan", "lemur", "meerkat",
	}
}
