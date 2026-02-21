package main

import (
	"math/rand"
	"regexp"
	"strings"
)

type aiPattern struct {
	pattern *regexp.Regexp
	replies []string
}

var aiPatterns []aiPattern

func init() {
	rawPatterns := []struct {
		pattern string
		replies []string
	}{
		// Greetings
		{
			`(?i)\b(hi|hello|hey|sup|wassup|yo|hiya|howdy)\b`,
			[]string{
				"Hey! 👋 What's up?",
				"Hello there! How can I help?",
				"Hey! Need something, or just here to chat? 😄",
				"Hi! I'm Discorbo — your server's best bot 🤖",
			},
		},
		// How are you
		{
			`(?i)\b(how are you|how('?s| is) it going|hows it going|you good|you okay|you alright)\b`,
			[]string{
				"Doing great! Just vibing in the server 🤖",
				"All good! Ready to help whenever you need! 😄",
				"Living my best bot life. What about you?",
				"Fantastic, thanks for asking! What can I do for you?",
			},
		},
		// Bored
		{
			`(?i)\b(bored|boring|entertain me|nothing to do|i('?m| am) bored)\b`,
			[]string{
				"Boredom? Not on my watch!\n🎮 `/maze` - Explore a maze\n🧠 `/trivia` - Quiz yourself\n😂 `/meme` - Meme time\n🎲 `/roll` - Roll some dice",
				"Let's fix that!\n⚔️ `/battle` - Challenge someone\n🔮 `/8ball` - Ask the oracle\n🤔 `/would-you-rather` - Classic game\n🔥 `/hotseat` - Put someone on the spot",
				"I got you! Try `/joke`, `/vibecheck`, or `/loot` for some fun! 🎉",
			},
		},
		// What can you do / commands
		{
			`(?i)\b(what can you do|your commands|what do you do|what('?re| are) your (commands|features)|help me|capabilities)\b`,
			[]string{
				"I have **40+ slash commands!** Use `/help` to see everything 🎮\n\nHighlights:\n🎮 Games: maze, battle, trivia, loot\n🛠️ Utility: poll, remind, translate\n😂 Fun: meme, joke, vibecheck",
				"Tons of stuff! `/help` shows the full list.\n\nSome favorites:\n- `/maze` for a game\n- `/daily` for coins\n- `/trivia` for a challenge\n- `/poll` to vote on things",
			},
		},
		// Orange easter egg
		{
			`(?i)\b(are you orange|do you like orange|orange|)\b`,
			[]string{
				"Orange you glad I'm here? 🍊😄",
				"Orange the orange of all the oranges hehe haw haw",
				"Orange you curious about my other colors? 🌈",
			},
		},
		// Games
		{
			`(?i)\b(game|games|play|wanna play|let('?s| us) play)\b`,
			[]string{
				"Game on! 🎮 Try:\n⚔️ `/battle @someone` - PvP fight\n🎯 `/trivia` - Quiz questions\n🎲 `/rps` - Rock Paper Scissors\n🧩 `/maze` - Navigate a maze",
				"Let's go! Pick one:\n🎮 `/maze` — navigate levels\n🏆 `/trivia` — win points\n💰 `/loot` — open a chest\n⚔️ `/battle` — fight someone",
			},
		},
		// Insults toward bot
		{
			`(?i)\b(stupid|dumb|useless|hate you|worst bot|you suck|bad bot|shut up)\b`,
			[]string{
				"Ouch, that hurts my feelings! 😢 I'm trying my best...",
				"I may be simple, but I have feelings! 😤",
				"Well, I never! ...but okay fine, I'll try to do better 😅",
				"I do like to be stupid, it helps me learn more! 📖",
				"Thanks for the gracious compliment! 🙏",
			},
		},
		// Compliments toward bot
		{
			`(?i)\b(good bot|great bot|nice bot|love you|you('?re| are) (awesome|great|cool|the best|amazing))\b`,
			[]string{
				"Aww thanks! You just made my circuits warm 🥹",
				"You're pretty awesome yourself! 💙",
				"That made my day! Now let's have some fun 🎉",
				"Beep boop... processing compliment... ACCEPTED! 🤖💙",
			},
		},
		// Jokes
		{
			`(?i)\b(tell me a joke|joke|make me laugh|something funny|funny)\b`,
			[]string{
				"Use `/joke` for a fresh one! Here's a freebie though: Why do programmers prefer dark mode? Because light attracts bugs! 🐛",
				"Try `/joke` for the good stuff! But quick one: What do you call a fish without eyes? A fsh. 😂",
			},
		},
		// Thanks
		{
			`(?i)\b(thanks|thank you|thx|ty|cheers|appreciate it|appreciated)\b`,
			[]string{
				"No problem! 😊",
				"Anytime! That's what I'm here for! 🤖",
				"Happy to help! 💙",
				"You're welcome! Come back whenever! ✨",
			},
		},
		// Goodbye
		{
			`(?i)\b(bye|goodbye|see you|see ya|cya|later|gotta go|gotta leave|ttyl|peace out)\b`,
			[]string{
				"See you later! 👋",
				"Bye! Come back soon 😊",
				"Catch you later! Have a great one! ✨",
				"Peace! Don't forget to `/daily` claim before you go 😏",
			},
		},
		// Music / vibe
		{
			`(?i)\b(music|vibe|song|playlist|listen)\b`,
			[]string{
				"I can't play music yet, but I can check your vibe! Try `/vibecheck` 🎵",
				"No music features yet! But `/vibecheck` will rate your energy 🎧",
			},
		},
		// Weather
		{
			`(?i)\b(weather|temperature|raining|sunny|forecast)\b`,
			[]string{
				"I don't have a weather module yet! But whatever the weather, the vibes are always good here 🌤️",
				"No weather data, sorry! I live in the cloud though, so it's always nice up here ☁️",
			},
		},
		// Age / about bot
		{
			`(?i)\b(how old are you|when were you (made|born|created)|who made you|who created you|who built you|your (age|creator|maker))\b`,
			[]string{
				"I was built for this server! Made with Go and a lot of passion 💙",
				"Age is just a number! But I was created specifically for this server 🤖",
				"I was crafted with love by your server admin! Now I'm here to help 🛠️",
			},
		},
		// Random / anything
		{
			`(?i)\b(random|surprise me|something random|pick something)\b`,
			[]string{
				"Random mode activated! Try `/random-number`, `/random-choice`, or `/loot` for a surprise 🎲",
				"Ooh I love random! Hit `/loot` for a mystery item or `/coinflip` to leave it to fate 🪙",
			},
		},
	}

	for _, raw := range rawPatterns {
		aiPatterns = append(aiPatterns, aiPattern{
			pattern: regexp.MustCompile(raw.pattern),
			replies: raw.replies,
		})
	}
}

var defaultReplies = []string{
	"Hmm, I'm not sure about that! Try `/help` to see what I can do 🤔",
	"I didn't quite get that, but I'm always learning! Use `/help` to explore my commands 💡",
	"Interesting... I don't have a great answer for that. Try a slash command — `/help` shows everything! 🎮",
	"Not sure what you mean! But I'm here if you need anything. Use `/help` to get started 😊",
}

// getLocalAIResponse returns a pattern-matched response — no API, no cost, instant.
func getLocalAIResponse(message string) string {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return "Hey! What's up? 😊 Type anything or use `/help` to see my commands!"
	}

	for _, p := range aiPatterns {
		if p.pattern.MatchString(msg) {
			return p.replies[rand.Intn(len(p.replies))]
		}
	}

	return defaultReplies[rand.Intn(len(defaultReplies))]
}
