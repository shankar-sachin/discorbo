package main

import (
	"fmt"
	"strings"
)

// ─── Card Rendering ────────────────────────────────────────────────────────────
// Compact Unicode card display: [A♠] [K♥] [10♦] [??]

// renderCard formats a single card as a bracketed Unicode string.
func renderCard(card string) string {
	if card == "" || card == "??" {
		return "`[??]`"
	}
	return fmt.Sprintf("`[%s]`", card)
}

// renderHand formats a slice of cards as a compact inline display.
func renderHand(cards []string) string {
	parts := make([]string, len(cards))
	for i, c := range cards {
		parts[i] = renderCard(c)
	}
	return strings.Join(parts, " ")
}

// renderHandHidden shows the first card and hides the rest.
func renderHandHidden(cards []string) string {
	if len(cards) == 0 {
		return renderCard("??")
	}
	parts := make([]string, len(cards))
	parts[0] = renderCard(cards[0])
	for i := 1; i < len(cards); i++ {
		parts[i] = renderCard("??")
	}
	return strings.Join(parts, " ")
}

// renderPokerHand shows cards with hold indicators underneath.
func renderPokerHand(cards []string, held []bool) string {
	line1 := renderHand(cards)
	if len(held) == 0 {
		return line1
	}
	parts := make([]string, len(cards))
	for i := range cards {
		if i < len(held) && held[i] {
			parts[i] = " ✅ "
		} else {
			parts[i] = " ⬜ "
		}
	}
	return line1 + "\n" + strings.Join(parts, " ")
}

// cardSuitEmoji returns a colored emoji for the suit.
func cardSuitEmoji(card string) string {
	if len(card) == 0 {
		return "🂠"
	}
	runes := []rune(card)
	suit := runes[len(runes)-1]
	switch suit {
	case '♠':
		return "♠️"
	case '♥':
		return "♥️"
	case '♦':
		return "♦️"
	case '♣':
		return "♣️"
	default:
		return "🂠"
	}
}

// isRedSuit returns true if the card's suit is red (hearts/diamonds).
func isRedSuit(card string) bool {
	if len(card) == 0 {
		return false
	}
	runes := []rune(card)
	suit := runes[len(runes)-1]
	return suit == '♥' || suit == '♦'
}

// ─── 2048 Board Rendering ──────────────────────────────────────────────────────
// Unicode box-drawing 2048 board with tile emojis.

// tileEmoji returns a colored square emoji based on tile value.
func tileEmoji(val int) string {
	switch val {
	case 0:
		return "⬛"
	case 2:
		return "⬜"
	case 4:
		return "🟫"
	case 8:
		return "🟧"
	case 16:
		return "🟠"
	case 32:
		return "🔴"
	case 64:
		return "🟥"
	case 128:
		return "🟡"
	case 256:
		return "🟨"
	case 512:
		return "🟢"
	case 1024:
		return "🟩"
	case 2048:
		return "💎"
	default:
		return "🟪" // > 2048
	}
}

// render2048Board creates a clean text-based 2048 board for Discord embeds.
func render2048Board(grid [4][4]int) string {
	var sb strings.Builder
	sb.WriteString("```\n")
	sb.WriteString("┌──────┬──────┬──────┬──────┐\n")
	for r := 0; r < 4; r++ {
		sb.WriteString("│")
		for c := 0; c < 4; c++ {
			v := grid[r][c]
			if v == 0 {
				sb.WriteString("      │")
			} else {
				sb.WriteString(fmt.Sprintf(" %4d │", v))
			}
		}
		sb.WriteString("\n")
		if r < 3 {
			sb.WriteString("├──────┼──────┼──────┼──────┤\n")
		}
	}
	sb.WriteString("└──────┴──────┴──────┴──────┘\n")
	sb.WriteString("```")
	return sb.String()
}

// render2048Emoji creates an emoji-based compact 2048 board.
func render2048Emoji(grid [4][4]int) string {
	var sb strings.Builder
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			sb.WriteString(tileEmoji(grid[r][c]))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ─── Slot Machine Rendering ────────────────────────────────────────────────────

var slotSymbols = []string{"🍒", "🍋", "🍊", "🍇", "🔔", "💎", "7️⃣"}

// renderSlotMachine shows a slot machine display.
func renderSlotMachine(reels [3]string, spinning bool) string {
	if spinning {
		return "🎰 **|** ❓ **|** ❓ **|** ❓ **|**"
	}
	return fmt.Sprintf("🎰 **|** %s **|** %s **|** %s **|**", reels[0], reels[1], reels[2])
}

// ─── Generic Grid Rendering ────────────────────────────────────────────────────

// renderGrid draws an arbitrary grid with custom cell renderer.
func renderGrid(rows, cols int, cellFn func(r, c int) string) string {
	var sb strings.Builder
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			sb.WriteString(cellFn(r, c))
		}
		if r < rows-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// ─── War Card Display ──────────────────────────────────────────────────────────

// renderWarCards shows two cards side by side for the War game.
func renderWarCards(playerCard, dealerCard string) string {
	return fmt.Sprintf(
		"**You** %s  ⚔️  %s **Dealer**",
		renderCard(playerCard), renderCard(dealerCard),
	)
}

// ─── Roulette Wheel ────────────────────────────────────────────────────────────

// renderRouletteResult shows the roulette outcome.
func renderRouletteResult(number int) string {
	var color string
	switch {
	case number == 0:
		color = "🟢"
	case isRouletteRed(number):
		color = "🔴"
	default:
		color = "⚫"
	}
	return fmt.Sprintf("%s **%d**", color, number)
}

func isRouletteRed(n int) bool {
	reds := map[int]bool{
		1: true, 3: true, 5: true, 7: true, 9: true, 12: true, 14: true,
		16: true, 18: true, 19: true, 21: true, 23: true, 25: true, 27: true,
		30: true, 32: true, 34: true, 36: true,
	}
	return reds[n]
}

// ─── Russian Roulette ──────────────────────────────────────────────────────────

// renderChambers shows a visual representation of chambers.
func renderChambers(total, current int) string {
	var sb strings.Builder
	for i := 0; i < total; i++ {
		if i < current {
			sb.WriteString("⚫ ") // fired (empty)
		} else if i == current {
			sb.WriteString("❓ ") // current chamber
		} else {
			sb.WriteString("🔘 ") // unfired
		}
	}
	return sb.String()
}

// ─── Progress Bars ─────────────────────────────────────────────────────────────

// renderProgressBar creates a text progress bar.
func renderProgressBar(current, max int, length int) string {
	if max <= 0 {
		max = 1
	}
	filled := (current * length) / max
	if filled > length {
		filled = length
	}
	if filled < 0 {
		filled = 0
	}
	empty := length - filled
	return fmt.Sprintf("[%s%s] %d/%d",
		strings.Repeat("█", filled),
		strings.Repeat("░", empty),
		current, max)
}

// renderHPBar creates a health bar for boss raids etc.
func renderHPBar(current, max int) string {
	return renderProgressBar(current, max, 20)
}

// renderXPBar creates an XP progress bar.
func renderXPBar(current, max int) string {
	return renderProgressBar(current, max, 15)
}

// ─── Connect4 Board ────────────────────────────────────────────────────────────

// renderConnect4Board renders a 7x6 Connect Four board.
func renderConnect4Board(board [6][7]int) string {
	var sb strings.Builder
	// Column numbers
	sb.WriteString("1️⃣2️⃣3️⃣4️⃣5️⃣6️⃣7️⃣\n")
	for r := 0; r < 6; r++ {
		for c := 0; c < 7; c++ {
			switch board[r][c] {
			case 1:
				sb.WriteString("🔴")
			case 2:
				sb.WriteString("🟡")
			default:
				sb.WriteString("⚫")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ─── Tic-Tac-Toe Board ────────────────────────────────────────────────────────

// renderTicTacToe renders a 3x3 tic-tac-toe board.
func renderTicTacToe(board [3][3]int) string {
	var sb strings.Builder
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			switch board[r][c] {
			case 1:
				sb.WriteString("❌")
			case 2:
				sb.WriteString("⭕")
			default:
				sb.WriteString("⬜")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ─── Wordle Rendering ──────────────────────────────────────────────────────────

// renderWordleRow shows a Wordle guess with colored squares.
// results: 0=wrong, 1=wrong position (yellow), 2=correct (green)
func renderWordleRow(guess string, results []int) string {
	var sb strings.Builder
	for i, r := range results {
		if i >= len(guess) {
			break
		}
		ch := strings.ToUpper(string(guess[i]))
		switch r {
		case 2:
			sb.WriteString("🟩")
		case 1:
			sb.WriteString("🟨")
		default:
			sb.WriteString("⬛")
		}
		sb.WriteString(ch + " ")
	}
	return sb.String()
}

// renderWordleBoard shows the full Wordle game board.
func renderWordleBoard(guesses []string, results [][]int) string {
	var sb strings.Builder
	for i, guess := range guesses {
		if i < len(results) {
			sb.WriteString(renderWordleRow(guess, results[i]))
		}
		sb.WriteString("\n")
	}
	// Empty rows
	for i := len(guesses); i < 6; i++ {
		sb.WriteString("⬛⬛⬛⬛⬛\n")
	}
	return sb.String()
}

// ─── Memory Game Rendering ─────────────────────────────────────────────────────

// renderMemoryBoard shows a 4x4 memory game grid.
func renderMemoryBoard(board [4][4]string, revealed [4][4]bool, matched [4][4]bool) string {
	var sb strings.Builder
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if matched[r][c] {
				sb.WriteString("✅")
			} else if revealed[r][c] {
				sb.WriteString(board[r][c])
			} else {
				sb.WriteString("❓")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
