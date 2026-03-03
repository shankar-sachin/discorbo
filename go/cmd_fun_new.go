package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ── ASCII Art ────────────────────────────────────────────────────────────────

// 5-line tall, 5-char wide block font for A-Z, 0-9, space
var asciiFont = map[rune][5]string{
	'A': {"  █  ", " █ █ ", "█████", "█   █", "█   █"},
	'B': {"████ ", "█   █", "████ ", "█   █", "████ "},
	'C': {" ████", "█    ", "█    ", "█    ", " ████"},
	'D': {"████ ", "█   █", "█   █", "█   █", "████ "},
	'E': {"█████", "█    ", "████ ", "█    ", "█████"},
	'F': {"█████", "█    ", "████ ", "█    ", "█    "},
	'G': {" ████", "█    ", "█  ██", "█   █", " ████"},
	'H': {"█   █", "█   █", "█████", "█   █", "█   █"},
	'I': {"█████", "  █  ", "  █  ", "  █  ", "█████"},
	'J': {"█████", "    █", "    █", "█   █", " ███ "},
	'K': {"█   █", "█  █ ", "███  ", "█  █ ", "█   █"},
	'L': {"█    ", "█    ", "█    ", "█    ", "█████"},
	'M': {"█   █", "██ ██", "█ █ █", "█   █", "█   █"},
	'N': {"█   █", "██  █", "█ █ █", "█  ██", "█   █"},
	'O': {" ███ ", "█   █", "█   █", "█   █", " ███ "},
	'P': {"████ ", "█   █", "████ ", "█    ", "█    "},
	'Q': {" ███ ", "█   █", "█ █ █", "█  █ ", " ██ █"},
	'R': {"████ ", "█   █", "████ ", "█  █ ", "█   █"},
	'S': {" ████", "█    ", " ███ ", "    █", "████ "},
	'T': {"█████", "  █  ", "  █  ", "  █  ", "  █  "},
	'U': {"█   █", "█   █", "█   █", "█   █", " ███ "},
	'V': {"█   █", "█   █", "█   █", " █ █ ", "  █  "},
	'W': {"█   █", "█   █", "█ █ █", "██ ██", "█   █"},
	'X': {"█   █", " █ █ ", "  █  ", " █ █ ", "█   █"},
	'Y': {"█   █", " █ █ ", "  █  ", "  █  ", "  █  "},
	'Z': {"█████", "   █ ", "  █  ", " █   ", "█████"},
	'0': {" ███ ", "█  ██", "█ █ █", "██  █", " ███ "},
	'1': {"  █  ", " ██  ", "  █  ", "  █  ", "█████"},
	'2': {" ███ ", "█   █", "  ██ ", " █   ", "█████"},
	'3': {"█████", "   █ ", " ███ ", "   █ ", "████ "},
	'4': {"█   █", "█   █", "█████", "    █", "    █"},
	'5': {"█████", "█    ", "████ ", "    █", "████ "},
	'6': {" ███ ", "█    ", "████ ", "█   █", " ███ "},
	'7': {"█████", "   █ ", "  █  ", " █   ", "█    "},
	'8': {" ███ ", "█   █", " ███ ", "█   █", " ███ "},
	'9': {" ███ ", "█   █", " ████", "    █", " ███ "},
	' ': {"     ", "     ", "     ", "     ", "     "},
}

func handleAsciiArt(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	text := strings.ToUpper(optionString(opts, "text", ""))
	if len(text) > 10 {
		text = text[:10]
	}
	if text == "" {
		respondEmbed(s, i, createErrorEmbed("Error", "Please provide some text."))
		return
	}
	lines := [5]string{}
	for _, ch := range text {
		glyph, ok := asciiFont[ch]
		if !ok {
			glyph = asciiFont[' ']
		}
		for row := 0; row < 5; row++ {
			lines[row] += glyph[row] + " "
		}
	}
	art := "```\n"
	for _, l := range lines {
		art += l + "\n"
	}
	art += "```"
	embed := createFunEmbed("🔤 ASCII Art", art)
	respondEmbed(s, i, embed)
}

// ── Countdown ────────────────────────────────────────────────────────────────

func handleCountdown(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	event := optionString(opts, "event", "Event")
	dateStr := optionString(opts, "date", "")
	target, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Invalid date format. Use YYYY-MM-DD."))
		return
	}
	now := time.Now().UTC()
	target = target.UTC()
	diff := target.Sub(now)
	if diff <= 0 {
		respondEmbed(s, i, createFunEmbed("⏰ Countdown", fmt.Sprintf("**%s** has already passed!", event)))
		return
	}
	days := int(diff.Hours()) / 24
	hours := int(diff.Hours()) % 24
	minutes := int(diff.Minutes()) % 60
	desc := fmt.Sprintf("⏳ **%s**\n\n📅 **Date:** %s\n\n**%d** days, **%d** hours, **%d** minutes remaining!",
		event, dateStr, days, hours, minutes)
	embed := createFunEmbed("⏰ Countdown", desc)
	respondEmbed(s, i, embed)
}

// ── Roast ────────────────────────────────────────────────────────────────────

var roastTemplates = []string{
	"**%s**, you're the reason God created the middle finger.",
	"**%s** is proof that even evolution makes mistakes sometimes.",
	"I'd roast **%s**, but I don't want to burn trash — it's bad for the environment.",
	"**%s** brings everyone so much joy… when they leave the room.",
	"**%s**, you're like a cloud. When you disappear, it's a beautiful day.",
	"If **%s** were any more basic, they'd be a pH 14.",
	"**%s**, I'd explain it to you, but I left my crayons at home.",
	"**%s** is the human equivalent of a participation trophy.",
	"**%s**, you're not stupid. You just have bad luck thinking.",
	"**%s** is like a software update — nobody wants you but you won't go away.",
	"**%s**, if laughter is the best medicine, your face must be curing the world.",
	"I bet **%s**'s WiFi disconnects out of shame.",
	"**%s**, you're the reason we have warning labels on everything.",
	"**%s** types at 10 WPM — and 8 of those are typos.",
	"**%s**, even autocorrect gave up on you.",
	"**%s** is the NPC energy the server didn't ask for.",
	"**%s**, your cooking is so bad, the smoke alarm cheers you on.",
	"If **%s** were a spice, they'd be flour.",
	"**%s**, you're living proof that brains aren't everything.",
	"**%s** tried to climb the ladder of success but took the escalator down.",
	"**%s**, even your imaginary friend found someone better.",
	"**%s** is about as useful as a screen door on a submarine.",
	"**%s**, you bring everyone together… in a group chat without you.",
	"**%s**, your secrets are always safe — nobody listens to you anyway.",
	"**%s** has the energy of an unread notification — easy to ignore.",
	"**%s**, you're the human version of a Monday morning.",
}

func handleRoast(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	u := optionUser(opts, "user")
	if u == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "User is required."))
		return
	}
	roast := fmt.Sprintf(roastTemplates[rand.Intn(len(roastTemplates))], u.Username)
	embed := createFunEmbed("🔥 Roast", roast+"\n\n*This is just for fun — no hard feelings!*")
	respondEmbed(s, i, embed)
}

// ── Compliment ───────────────────────────────────────────────────────────────

var complimentTemplates = []string{
	"**%s**, you light up every server you join! ✨",
	"**%s** is the friend everyone deserves but few are lucky to have.",
	"**%s**, your vibe is immaculate and your energy is unmatched.",
	"The world is a better place with **%s** in it.",
	"**%s**, if kindness were currency, you'd be a billionaire.",
	"**%s** has the kind of smile that makes bad days disappear.",
	"**%s**, you're proof that good people still exist.",
	"**%s** is the main character and everyone knows it.",
	"**%s**, your creativity and intelligence are truly inspiring.",
	"If there were more people like **%s**, the world would be at peace.",
	"**%s**, you make the impossible look effortless.",
	"**%s** could brighten even the darkest Discord server.",
	"**%s**, you're like a rare legendary drop — one of a kind.",
	"**%s** has big protagonist energy and the plot armor to match.",
	"**%s**, your presence alone raises the server's vibe by 200%%.",
	"**%s** is the type of person who makes strangers smile.",
	"**%s**, you're not just a star — you're the whole constellation.",
	"**%s** has more talent in one pinky than most have in their whole body.",
	"**%s**, the way you carry yourself is genuinely admirable.",
	"**%s** could solve world hunger just by being in the room.",
	"**%s**, you're the reason someone believes in good people today.",
	"**%s** radiates confidence and kindness in equal measure.",
	"**%s**, talking to you is like finding a $20 bill in your pocket.",
	"**%s** is a limited edition — no copies, no reprints.",
	"**%s**, you have the heart of a champion and the soul of an artist.",
	"**%s** makes every conversation worth having.",
}

func handleCompliment(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	u := optionUser(opts, "user")
	if u == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "User is required."))
		return
	}
	comp := fmt.Sprintf(complimentTemplates[rand.Intn(len(complimentTemplates))], u.Username)
	embed := createFunEmbed("💖 Compliment", comp)
	respondEmbed(s, i, embed)
}

// ── Truth or Dare ────────────────────────────────────────────────────────────

var truthQuestions = []string{
	"What's the most embarrassing thing you've done online?",
	"What's your most unpopular opinion?",
	"What's the last lie you told?",
	"What's your guilty pleasure song?",
	"If you could swap lives with someone in this server, who?",
	"What's the weirdest thing you've Googled?",
	"What's your biggest pet peeve?",
	"Have you ever pretended to like a gift?",
	"What's the longest you've binged a show?",
	"What's the most childish thing you still do?",
	"What secret talent do you have?",
	"What's the worst advice you've ever given?",
	"Have you ever blamed someone else for something you did?",
	"What's the most embarrassing item in your room right now?",
	"If your search history was made public, how screwed are you?",
	"What's a trend you secretly enjoy?",
	"What's the pettiest thing you've ever done?",
	"Have you ever stalked someone's social media?",
	"What's the worst haircut you've ever had?",
	"What's your most irrational fear?",
	"What's the most useless skill you have?",
}

var darePrompts = []string{
	"Change your nickname to 'I Lost a Dare' for 10 minutes.",
	"Send a voice message saying 'I am a banana' dramatically.",
	"Type your next 3 messages in ALL CAPS.",
	"Let someone else send a message from your account.",
	"Post your most recently used emoji 20 times.",
	"Compliment everyone in the voice channel right now.",
	"Use only song lyrics to respond for the next 5 minutes.",
	"Send a selfie with a silly face.",
	"Describe your crush using only emojis.",
	"Share the last photo in your camera roll.",
	"Talk in third person for the next 5 minutes.",
	"Send a DM to the last person you messaged saying 'I believe in you'.",
	"React to the next 10 messages with 🤡.",
	"Try to make someone in the server laugh within 2 minutes.",
	"Share your screen time report.",
	"Type a paragraph with your eyes closed.",
	"Sing a song in the voice channel.",
	"Let the group pick your profile picture for 1 hour.",
	"Do your best impression of a server admin.",
	"Speak in an accent for the next 5 minutes.",
	"Post an embarrassing photo from your gallery.",
}

func handleTruthOrDare(s *discordgo.Session, i *discordgo.InteractionCreate) {
	isTruth := rand.Intn(2) == 0
	var title, prompt string
	if isTruth {
		title = "🤔 Truth"
		prompt = truthQuestions[rand.Intn(len(truthQuestions))]
	} else {
		title = "😈 Dare"
		prompt = darePrompts[rand.Intn(len(darePrompts))]
	}
	embed := createFunEmbed(title, prompt)
	row := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "🔄 Another!",
				Style:    discordgo.PrimaryButton,
				CustomID: "truthordare_reroll",
			},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{row},
		},
	})
}

// handleTruthOrDareButton handles the reroll button press.
func handleTruthOrDareButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	isTruth := rand.Intn(2) == 0
	var title, prompt string
	if isTruth {
		title = "🤔 Truth"
		prompt = truthQuestions[rand.Intn(len(truthQuestions))]
	} else {
		title = "😈 Dare"
		prompt = darePrompts[rand.Intn(len(darePrompts))]
	}
	embed := createFunEmbed(title, prompt)
	row := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "🔄 Another!",
				Style:    discordgo.PrimaryButton,
				CustomID: "truthordare_reroll",
			},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{row},
		},
	})
}

// ── This or That ─────────────────────────────────────────────────────────────

var thisOrThatPairs = [][2]string{
	{"Cats", "Dogs"},
	{"Pizza", "Burgers"},
	{"Morning person", "Night owl"},
	{"Movies", "TV Shows"},
	{"Summer", "Winter"},
	{"Coffee", "Tea"},
	{"Beach", "Mountains"},
	{"Books", "Podcasts"},
	{"Invisibility", "Flying"},
	{"Android", "iPhone"},
	{"Marvel", "DC"},
	{"Pancakes", "Waffles"},
	{"Ice cream", "Cake"},
	{"Texting", "Calling"},
	{"City life", "Country life"},
	{"Sweet", "Savory"},
	{"Past", "Future"},
	{"Spotify", "Apple Music"},
	{"Early bird", "Night owl"},
	{"Messy room", "Clean room"},
	{"Comedy", "Horror"},
}

func handleThisOrThat(s *discordgo.Session, i *discordgo.InteractionCreate) {
	pair := thisOrThatPairs[rand.Intn(len(thisOrThatPairs))]
	desc := fmt.Sprintf("**%s** 🔵  vs  🔴 **%s**\n\nReact or reply with your pick!", pair[0], pair[1])
	embed := createFunEmbed("⚡ This or That", desc)
	respondEmbed(s, i, embed)
}

// ── Fake Tweet ───────────────────────────────────────────────────────────────

func handleFakeTweet(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	u := optionUser(opts, "user")
	if u == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "User is required."))
		return
	}
	text := optionString(opts, "text", "Hello world!")
	likes := rand.Intn(99999) + 1
	retweets := rand.Intn(25000) + 1
	replies := rand.Intn(5000) + 1

	embed := &discordgo.MessageEmbed{
		Color: 0x1DA1F2,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    fmt.Sprintf("%s @%s", u.Username, u.Username),
			IconURL: u.AvatarURL("64"),
		},
		Description: text,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "💬 Replies", Value: formatCount(replies), Inline: true},
			{Name: "🔁 Retweets", Value: formatCount(retweets), Inline: true},
			{Name: "❤️ Likes", Value: formatCount(likes), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Twitter • " + time.Now().Format("3:04 PM · Jan 2, 2006"),
		},
	}
	respondEmbed(s, i, embed)
}

func formatCount(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// ── Emoji Mix ────────────────────────────────────────────────────────────────

var emojiDescriptions = map[string]string{
	"😀": "happy face", "😂": "laughing face", "😍": "heart eyes",
	"🤔": "thinking face", "😎": "cool face", "🥺": "pleading face",
	"😡": "angry face", "💀": "skull", "🔥": "fire",
	"❤️": "heart", "⭐": "star", "🌈": "rainbow",
	"🎉": "party", "💎": "diamond", "🦄": "unicorn",
	"🐱": "cat", "🐶": "dog", "🍕": "pizza",
	"🎸": "guitar", "🚀": "rocket", "👻": "ghost",
	"🤖": "robot", "🌊": "wave", "⚡": "lightning",
	"🎵": "music", "🍦": "ice cream", "🌙": "moon",
	"☀️": "sun", "🦋": "butterfly", "🎮": "gaming",
}

var mixTemplates = []string{
	"A %s infused with the power of %s",
	"Imagine a %s but it radiates pure %s energy",
	"The legendary fusion: %s meets %s in an epic combo",
	"A %s that secretly dreams of being a %s",
	"When %s and %s had a baby, this was born",
	"%s energy with a hint of %s — chaotic but beautiful",
	"A %s wearing a %s costume at a party",
	"The forbidden crossover: %s × %s",
}

func handleEmojiMix(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	e1 := optionString(opts, "emoji1", "😀")
	e2 := optionString(opts, "emoji2", "🔥")
	d1, ok1 := emojiDescriptions[e1]
	if !ok1 {
		d1 = e1
	}
	d2, ok2 := emojiDescriptions[e2]
	if !ok2 {
		d2 = e2
	}
	tmpl := mixTemplates[rand.Intn(len(mixTemplates))]
	result := fmt.Sprintf(tmpl, d1, d2)
	desc := fmt.Sprintf("%s + %s = ?\n\n**Result:** %s", e1, e2, result)
	embed := createFunEmbed("🧬 Emoji Mix", desc)
	respondEmbed(s, i, embed)
}

// ── Fortune ──────────────────────────────────────────────────────────────────

var fortuneMessages = []string{
	"🥠 A surprise awaits you in the near future.",
	"🥠 Your hard work will soon pay off.",
	"🥠 An unexpected friend will bring you joy.",
	"🥠 The best is yet to come — be patient.",
	"🥠 A great adventure is just around the corner.",
	"🥠 Good things come to those who meme.",
	"🥠 Your code will compile on the first try... eventually.",
	"🥠 The stars say you should take a nap.",
	"🥠 You will find what you seek in the last place you look.",
	"🥠 Today's struggles are tomorrow's strengths.",
	"🥠 A new friendship will blossom in an unexpected place.",
	"🥠 Fortune favors the bold — and the caffeinated.",
	"🥠 Something wonderful is about to happen. Stay alert.",
	"🥠 You will master a new skill sooner than you think.",
	"🥠 An old dream will find new life this week.",
	"🥠 Trust the process. Also, drink water.",
	"🥠 Your next decision will be the right one. Probably.",
	"🥠 The universe has WiFi, and you're connected.",
	"🥠 You will receive a compliment that makes your day.",
	"🥠 A plot twist in your story will turn out to be a blessing.",
	"🥠 Patience is bitter, but its fruit is sweet — and probably pizza.",
	"🥠 The person reading this is going places. Good ones.",
	"🥠 A lost item will be found in an obvious spot.",
	"🥠 Your creative energy is at an all-time high. Use it!",
	"🥠 Someone is thinking about you right now. Smile!",
	"🥠 An opportunity will knock twice. Answer the door.",
	"🥠 Your next meal will be unexpectedly delicious.",
	"🥠 A message you send today will change someone's mood.",
	"🥠 You will laugh so hard you cry before the week ends.",
	"🥠 The answer to your question is: yes, but with style.",
	"🥠 Your future self is proud of the choices you're making today.",
}

func handleFortune(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fortune := fortuneMessages[rand.Intn(len(fortuneMessages))]
	embed := createFunEmbed("🥠 Fortune Cookie", fortune)
	respondEmbed(s, i, embed)
}
