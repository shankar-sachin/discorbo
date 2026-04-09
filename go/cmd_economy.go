package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Shop command handler
func handleShop(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}

	sub, subOpts := getSubcommand(opts)
	if sub == "" {
		sub = "list"
	}

	switch sub {
	case "list":
		handleShopList(s, i, subOpts, user)
	case "buy":
		handleShopBuy(s, i, subOpts, user)
	case "sell":
		handleShopSell(s, i, subOpts, user)
	default:
		respondError(s, i, "Error", "Unknown subcommand.")
	}
}

func handleShopList(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption, user *discordgo.User) {
	category := strings.ToLower(optionString(opts, "category", "all"))

	catalog := map[string]shopItem{}
	if err := readData("shop-catalog.json", &catalog); err != nil {
		respondError(s, i, "Error", "Failed to load shop catalog.")
		return
	}

	// Filter by category
	items := []shopItem{}
	for _, item := range catalog {
		if category == "all" || item.Category == category {
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		respondError(s, i, "Error", "No items found in this category.")
		return
	}

	// Sort by price
	sort.Slice(items, func(a, b int) bool {
		return items[a].Price < items[b].Price
	})

	// Build embed
	lines := []string{}
	for _, item := range items {
		ownershipInfo := ""
		if item.MaxOwned > 0 {
			ownershipInfo = fmt.Sprintf(" (Max: %d)", item.MaxOwned)
		}
		lines = append(lines, fmt.Sprintf("%s **%s** - %s%s\n*%s*\nID: `%s`",
			item.Emoji, item.Name, coinDisplay(item.Price), ownershipInfo, item.Description, item.ID))
	}

	// Split into chunks if too long
	description := strings.Join(lines, "\n\n")
	if len(description) > 2000 {
		description = description[:1997] + "..."
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🛒 Shop Catalog",
		Description: description,
		Color:       ColorEconomy,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer:      embedFooter("💰 Economy"),
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: "💡 How to Buy", Value: "Use `/economy shop buy <item_id>`", Inline: false,
	})

	respondEmbed(s, i, embed)
}

func handleShopBuy(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption, user *discordgo.User) {
	itemID := strings.TrimSpace(optionString(opts, "item_id", ""))
	quantity := int(optionInt(opts, "quantity", 1))

	if itemID == "" {
		respondError(s, i, "Error", "Item ID is required.")
		return
	}
	if quantity < 1 {
		respondError(s, i, "Error", "Quantity must be at least 1.")
		return
	}

	// Load catalog
	catalog := map[string]shopItem{}
	if err := readData("shop-catalog.json", &catalog); err != nil {
		respondError(s, i, "Error", "Failed to load shop catalog.")
		return
	}

	item, exists := catalog[itemID]
	if !exists {
		respondError(s, i, "Error", fmt.Sprintf("Item `%s` not found in shop.", itemID))
		return
	}

	// Load user economy data
	economyUsers := map[string]economyUser{}
	_ = readData("economy-users.json", &economyUsers)
	ecoUser := economyUsers[user.ID]
	if ecoUser.Username == "" {
		ecoUser = economyUser{
			Username:     user.Username,
			Coins:        0,
			Inventory:    []inventoryItem{},
			ActiveBoosts: []activeBoost{},
			TradeHistory: []tradeRecord{},
		}
	}
	ecoUser.Username = user.Username

	// Check max owned
	if item.MaxOwned > 0 {
		owned := 0
		for _, inv := range ecoUser.Inventory {
			if inv.ItemID == itemID {
				owned = inv.Quantity
				break
			}
		}
		if owned+quantity > item.MaxOwned {
			respondError(s, i, "Error", fmt.Sprintf("You can only own %d of this item. You already have %d.", item.MaxOwned, owned))
			return
		}
	}

	totalCost := item.Price * quantity
	if ecoUser.Coins < totalCost {
		respondError(s, i, "Error", fmt.Sprintf("Not enough coins. Need %s, have %s.", coinDisplay(totalCost), coinDisplay(ecoUser.Coins)))
		return
	}

	// Process purchase
	ecoUser.Coins -= totalCost
	ecoUser.TotalSpent += totalCost

	// Add to inventory
	found := false
	for idx, inv := range ecoUser.Inventory {
		if inv.ItemID == itemID {
			ecoUser.Inventory[idx].Quantity += quantity
			found = true
			break
		}
	}
	if !found {
		ecoUser.Inventory = append(ecoUser.Inventory, inventoryItem{
			ItemID:      itemID,
			Quantity:    quantity,
			PurchasedAt: time.Now().UnixMilli(),
		})
	}

	economyUsers[user.ID] = ecoUser
	_ = writeData("economy-users.json", economyUsers)

	// Log transaction
	logTransaction(user.ID, "purchase", itemID, -totalCost, "", "")

	embed := createSuccessEmbed("Purchase Complete",
		fmt.Sprintf("%s Bought **%dx %s** for %s\nRemaining balance: %s",
			item.Emoji, quantity, item.Name, coinDisplay(totalCost), coinDisplay(ecoUser.Coins)))
	respondEmbed(s, i, embed)
}

func handleShopSell(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption, user *discordgo.User) {
	itemID := strings.TrimSpace(optionString(opts, "item_id", ""))
	quantity := int(optionInt(opts, "quantity", 1))

	if itemID == "" {
		respondError(s, i, "Error", "Item ID is required.")
		return
	}
	if quantity < 1 {
		respondError(s, i, "Error", "Quantity must be at least 1.")
		return
	}

	// Load catalog
	catalog := map[string]shopItem{}
	if err := readData("shop-catalog.json", &catalog); err != nil {
		respondError(s, i, "Error", "Failed to load shop catalog.")
		return
	}

	item, exists := catalog[itemID]
	if !exists {
		respondError(s, i, "Error", fmt.Sprintf("Item `%s` not found.", itemID))
		return
	}

	// Load user economy data
	economyUsers := map[string]economyUser{}
	_ = readData("economy-users.json", &economyUsers)
	ecoUser := economyUsers[user.ID]
	if ecoUser.Username == "" {
		respondError(s, i, "Error", "You don't have any items to sell.")
		return
	}

	// Check inventory
	foundIdx := -1
	for idx, inv := range ecoUser.Inventory {
		if inv.ItemID == itemID {
			foundIdx = idx
			break
		}
	}

	if foundIdx == -1 {
		respondError(s, i, "Error", fmt.Sprintf("You don't own any %s.", item.Name))
		return
	}

	if ecoUser.Inventory[foundIdx].Quantity < quantity {
		respondError(s, i, "Error", fmt.Sprintf("You only have %d of this item.", ecoUser.Inventory[foundIdx].Quantity))
		return
	}

	// Calculate refund (50%)
	refund := (item.Price * quantity) / 2

	// Process sale
	ecoUser.Coins += refund
	ecoUser.Inventory[foundIdx].Quantity -= quantity

	// Remove from inventory if quantity is 0
	if ecoUser.Inventory[foundIdx].Quantity == 0 {
		ecoUser.Inventory = append(ecoUser.Inventory[:foundIdx], ecoUser.Inventory[foundIdx+1:]...)
	}

	economyUsers[user.ID] = ecoUser
	_ = writeData("economy-users.json", economyUsers)

	// Log transaction
	logTransaction(user.ID, "sell", itemID, refund, "", "")

	embed := createSuccessEmbed("Sold Item",
		fmt.Sprintf("Sold **%dx %s** for %s (50%% refund)\nNew balance: %s",
			quantity, item.Name, coinDisplay(refund), coinDisplay(ecoUser.Coins)))
	respondEmbed(s, i, embed)
}

// Inventory command handler
func handleInventory(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}

	sub, subOpts := getSubcommand(opts)
	if sub == "" {
		sub = "view"
	}

	switch sub {
	case "view":
		handleInventoryView(s, i, user)
	case "use":
		handleInventoryUse(s, i, subOpts, user)
	case "info":
		handleInventoryInfo(s, i, subOpts, user)
	default:
		respondError(s, i, "Error", "Unknown subcommand.")
	}
}

func handleInventoryView(s *discordgo.Session, i *discordgo.InteractionCreate, user *discordgo.User) {
	economyUsers := map[string]economyUser{}
	_ = readData("economy-users.json", &economyUsers)
	ecoUser := economyUsers[user.ID]

	if len(ecoUser.Inventory) == 0 {
		respondError(s, i, "Error", "Your inventory is empty. Visit /shop to buy items!")
		return
	}

	catalog := map[string]shopItem{}
	_ = readData("shop-catalog.json", &catalog)

	lines := []string{}
	for _, inv := range ecoUser.Inventory {
		item := catalog[inv.ItemID]
		lines = append(lines, fmt.Sprintf("%s **%s** x%d\n`%s`", item.Emoji, item.Name, inv.Quantity, inv.ItemID))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📦 Your Inventory",
		Description: strings.Join(lines, "\n\n"),
		Color:       ColorBlue,
		Footer: &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("%d items  •  " + botVersion, len(ecoUser.Inventory))},
	}

	respondEmbed(s, i, embed)
}

func handleInventoryUse(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption, user *discordgo.User) {
	itemID := strings.TrimSpace(optionString(opts, "item_id", ""))
	if itemID == "" {
		respondError(s, i, "Error", "Item ID is required.")
		return
	}

	catalog := map[string]shopItem{}
	_ = readData("shop-catalog.json", &catalog)
	item, exists := catalog[itemID]
	if !exists {
		respondError(s, i, "Error", fmt.Sprintf("Item `%s` not found.", itemID))
		return
	}

	economyUsers := map[string]economyUser{}
	_ = readData("economy-users.json", &economyUsers)
	ecoUser := economyUsers[user.ID]

	// Check if user owns the item
	foundIdx := -1
	for idx, inv := range ecoUser.Inventory {
		if inv.ItemID == itemID {
			foundIdx = idx
			break
		}
	}

	if foundIdx == -1 {
		respondError(s, i, "Error", fmt.Sprintf("You don't own any %s.", item.Name))
		return
	}

	// Use the item based on type
	if item.Category == "boost" {
		// Apply boost
		duration := int64(24 * time.Hour / time.Millisecond)
		if item.BoostType == "xp_multiplier" {
			duration = int64(12 * time.Hour / time.Millisecond)
		} else if item.BoostType == "global_coin_multiplier" {
			duration = int64(6 * time.Hour / time.Millisecond)
		}

		expiresAt := time.Now().UnixMilli() + duration

		// Check if boost already active
		for idx, boost := range ecoUser.ActiveBoosts {
			if boost.BoostType == item.BoostType && boost.ExpiresAt > time.Now().UnixMilli() {
				// Extend duration
				ecoUser.ActiveBoosts[idx].ExpiresAt = expiresAt
				ecoUser.Inventory[foundIdx].Quantity--
				if ecoUser.Inventory[foundIdx].Quantity == 0 {
					ecoUser.Inventory = append(ecoUser.Inventory[:foundIdx], ecoUser.Inventory[foundIdx+1:]...)
				}
				economyUsers[user.ID] = ecoUser
				_ = writeData("economy-users.json", economyUsers)

				embed := createSuccessEmbed("Boost Extended",
					fmt.Sprintf("%s **%s** boost extended!\nExpires: <t:%d:R>",
						item.Emoji, item.Name, expiresAt/1000))
				respondEmbed(s, i, embed)
				return
			}
		}

		// Add new boost
		ecoUser.ActiveBoosts = append(ecoUser.ActiveBoosts, activeBoost{
			BoostType:  item.BoostType,
			Multiplier: item.BoostValue,
			ExpiresAt:  expiresAt,
		})

		ecoUser.Inventory[foundIdx].Quantity--
		if ecoUser.Inventory[foundIdx].Quantity == 0 {
			ecoUser.Inventory = append(ecoUser.Inventory[:foundIdx], ecoUser.Inventory[foundIdx+1:]...)
		}

		economyUsers[user.ID] = ecoUser
		_ = writeData("economy-users.json", economyUsers)

		embed := createSuccessEmbed("Boost Activated",
			fmt.Sprintf("%s **%s** activated!\n%.1fx %s\nExpires: <t:%d:R>",
				item.Emoji, item.Name, item.BoostValue, item.BoostType, expiresAt/1000))
		respondEmbed(s, i, embed)

	} else if item.Category == "consumable" && item.ID == "mystery_box" {
		// Open mystery box
		rewards := []string{
			"50 coins",
			"100 coins",
			"200 coins",
			"500 coins (RARE!)",
		}
		reward := rewards[rand.Intn(len(rewards))]

		coinsGained := 50
		if strings.HasPrefix(reward, "100") {
			coinsGained = 100
		} else if strings.HasPrefix(reward, "200") {
			coinsGained = 200
		} else if strings.HasPrefix(reward, "500") {
			coinsGained = 500
		}

		ecoUser.Coins += coinsGained
		ecoUser.Inventory[foundIdx].Quantity--
		if ecoUser.Inventory[foundIdx].Quantity == 0 {
			ecoUser.Inventory = append(ecoUser.Inventory[:foundIdx], ecoUser.Inventory[foundIdx+1:]...)
		}

		economyUsers[user.ID] = ecoUser
		_ = writeData("economy-users.json", economyUsers)

		embed := createSuccessEmbed("Mystery Box Opened!",
			fmt.Sprintf("🎁 You received: **%s**\nNew balance: %s", reward, coinDisplay(ecoUser.Coins)))
		respondEmbed(s, i, embed)

	} else {
		respondError(s, i, "Error", "This item cannot be used directly. It may be a collectible or passive boost.")
	}
}

func handleInventoryInfo(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption, user *discordgo.User) {
	itemID := strings.TrimSpace(optionString(opts, "item_id", ""))
	if itemID == "" {
		respondError(s, i, "Error", "Item ID is required.")
		return
	}

	catalog := map[string]shopItem{}
	_ = readData("shop-catalog.json", &catalog)
	item, exists := catalog[itemID]
	if !exists {
		respondError(s, i, "Error", fmt.Sprintf("Item `%s` not found.", itemID))
		return
	}

	economyUsers := map[string]economyUser{}
	_ = readData("economy-users.json", &economyUsers)
	ecoUser := economyUsers[user.ID]

	owned := 0
	for _, inv := range ecoUser.Inventory {
		if inv.ItemID == itemID {
			owned = inv.Quantity
			break
		}
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Price", Value: coinDisplay(item.Price), Inline: true},
		{Name: "Category", Value: item.Category, Inline: true},
		{Name: "You Own", Value: fmt.Sprintf("%d", owned), Inline: true},
	}

	if item.MaxOwned > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Max Owned",
			Value:  fmt.Sprintf("%d", item.MaxOwned),
			Inline: true,
		})
	}

	if item.BoostType != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Boost Effect",
			Value:  fmt.Sprintf("%.1fx %s", item.BoostValue, item.BoostType),
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s %s", item.Emoji, item.Name),
		Description: item.Description,
		Color:       ColorBlue,
		Fields:      fields,
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("ID: %s  •  " + botVersion, item.ID)},
	}

	respondEmbed(s, i, embed)
}

// Balance command handler
func handleBalance(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}

	economyUsers := map[string]economyUser{}
	_ = readData("economy-users.json", &economyUsers)
	ecoUser := economyUsers[user.ID]

	// Clean up expired boosts
	activeBoosts := []activeBoost{}
	now := time.Now().UnixMilli()
	for _, boost := range ecoUser.ActiveBoosts {
		if boost.ExpiresAt > now {
			activeBoosts = append(activeBoosts, boost)
		}
	}
	ecoUser.ActiveBoosts = activeBoosts

	fields := []*discordgo.MessageEmbedField{
		{Name: "💰 Balance", Value: coinDisplay(ecoUser.Coins), Inline: true},
		{Name: "📈 Total Earned", Value: coinDisplay(ecoUser.TotalEarned), Inline: true},
		{Name: "📉 Total Spent", Value: coinDisplay(ecoUser.TotalSpent), Inline: true},
		{Name: "📦 Inventory", Value: fmt.Sprintf("%d item(s)", len(ecoUser.Inventory)), Inline: true},
	}

	if len(ecoUser.ActiveBoosts) > 0 {
		boostLines := []string{}
		for _, boost := range ecoUser.ActiveBoosts {
			boostLines = append(boostLines, fmt.Sprintf("⚡ **%.1fx** %s — expires <t:%d:R>",
				boost.Multiplier, boost.BoostType, boost.ExpiresAt/1000))
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "🔥 Active Boosts",
			Value:  strings.Join(boostLines, "\n"),
			Inline: false,
		})
	}

	embed := richEmbed(richEmbedOpts{
		Title:        fmt.Sprintf("💰 %s's Wallet", user.Username),
		Color:        ColorEconomy,
		Category:     "💰 Economy",
		Fields:       fields,
		ThumbnailURL: userAvatar(user),
		AuthorName:   user.Username,
		AuthorIcon:   userAvatar(user),
	})

	respondEmbed(s, i, embed)
}

// Trade command handler
func handleTrade(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}

	sub, subOpts := getSubcommand(opts)
	if sub == "" {
		sub = "offer"
	}

	switch sub {
	case "offer":
		handleTradeOffer(s, i, subOpts, user)
	default:
		respondError(s, i, "Error", "Unknown subcommand.")
	}
}

func handleTradeOffer(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption, user *discordgo.User) {
	targetUser := optionUser(opts, "user")
	if targetUser == nil {
		respondError(s, i, "Error", "You must specify a user to trade with.")
		return
	}
	if targetUser.Bot {
		respondError(s, i, "Error", "You cannot trade with bots.")
		return
	}
	if targetUser.ID == user.ID {
		respondError(s, i, "Error", "You cannot trade with yourself.")
		return
	}

	// Create trade session
	tradeID := fmt.Sprintf("%s_%s_%d", user.ID, targetUser.ID, time.Now().UnixMilli())

	embed := &discordgo.MessageEmbed{
		Title:       "🤝 Trade Request",
		Description: fmt.Sprintf("%s wants to trade with you!\n\nClick Accept to start trading.", user.Username),
		Color:       ColorPurple,
		Footer:      embedFooter("💰 Economy"),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	buttons := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Accept",
				Style:    discordgo.SuccessButton,
				CustomID: fmt.Sprintf("trade_accept_%s", tradeID),
			},
			discordgo.Button{
				Label:    "Decline",
				Style:    discordgo.DangerButton,
				CustomID: fmt.Sprintf("trade_decline_%s", tradeID),
			},
		},
	}

	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{buttons},
		},
	}

	if err := s.InteractionRespond(i.Interaction, resp); err != nil {
		return
	}

	// Get message ID for session tracking
	msg, _ := s.InteractionResponse(i.Interaction)
	if msg != nil {
		tradeMu.Lock()
		tradeSessions[msg.ID] = &tradeSession{
			TradeID:     tradeID,
			InitiatorID: user.ID,
			TargetID:    targetUser.ID,
			InitOffer:   tradeOffer{Coins: 0, ItemIDs: []string{}},
			TargetOffer: tradeOffer{Coins: 0, ItemIDs: []string{}},
			ExpiresAt:   time.Now().Add(5 * time.Minute).UnixMilli(),
			MessageID:   msg.ID,
			ChannelID:   i.ChannelID,
		}
		tradeMu.Unlock()

		// Auto-cleanup after 5 minutes
		go func(msgID string) {
			time.Sleep(5 * time.Minute)
			tradeMu.Lock()
			if sess, ok := tradeSessions[msgID]; ok {
				if !sess.InitConfirmed || !sess.TargetConfirmed {
					delete(tradeSessions, msgID)
				}
			}
			tradeMu.Unlock()
		}(msg.ID)
	}
}

// Economy admin command handler
func handleEconomyAdmin(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	// Check admin permission
	if i.Member == nil || (i.Member.Permissions&discordgo.PermissionAdministrator) == 0 {
		respondError(s, i, "Error", "This command requires Administrator permission.")
		return
	}

	sub, subOpts := getSubcommand(opts)
	if sub == "" {
		respondError(s, i, "Error", "Subcommand required.")
		return
	}

	admin := interactionUser(i)
	if admin == nil {
		respondError(s, i, "Error", "Unable to identify admin.")
		return
	}

	switch sub {
	case "grant":
		targetUser := optionUser(subOpts, "user")
		coins := int(optionInt(subOpts, "coins", 0))
		reason := optionString(subOpts, "reason", "Admin grant")

		if targetUser == nil || coins <= 0 {
			respondError(s, i, "Error", "Valid user and positive coin amount required.")
			return
		}

		economyUsers := map[string]economyUser{}
		_ = readData("economy-users.json", &economyUsers)
		eu := economyUsers[targetUser.ID]
		eu.Username = targetUser.Username
		eu.Coins += coins
		eu.TotalEarned += coins
		economyUsers[targetUser.ID] = eu
		_ = writeData("economy-users.json", economyUsers)

		logTransaction(targetUser.ID, "admin_grant", "", coins, admin.ID, reason)

		embed := createSuccessEmbed("Coins Granted",
			fmt.Sprintf("Granted %s to %s\nReason: %s", coinDisplay(coins), targetUser.Username, reason))
		respondEmbed(s, i, embed)

	case "take":
		targetUser := optionUser(subOpts, "user")
		coins := int(optionInt(subOpts, "coins", 0))
		reason := optionString(subOpts, "reason", "Admin removal")

		if targetUser == nil || coins <= 0 {
			respondError(s, i, "Error", "Valid user and positive coin amount required.")
			return
		}

		economyUsers := map[string]economyUser{}
		_ = readData("economy-users.json", &economyUsers)
		eu := economyUsers[targetUser.ID]
		eu.Coins = max(0, eu.Coins-coins)
		economyUsers[targetUser.ID] = eu
		_ = writeData("economy-users.json", economyUsers)

		logTransaction(targetUser.ID, "admin_take", "", -coins, admin.ID, reason)

		embed := createSuccessEmbed("Coins Removed",
			fmt.Sprintf("Removed %s from %s\nReason: %s", coinDisplay(coins), targetUser.Username, reason))
		respondEmbed(s, i, embed)

	case "transactions":
		targetUser := optionUser(subOpts, "user")
		transactions := []transactionLog{}
		_ = readData("transactions.json", &transactions)

		filtered := []transactionLog{}
		for _, t := range transactions {
			if targetUser == nil || t.UserID == targetUser.ID {
				filtered = append(filtered, t)
			}
		}

		if len(filtered) == 0 {
			respondError(s, i, "Error", "No transactions found.")
			return
		}

		// Show last 10
		if len(filtered) > 10 {
			filtered = filtered[len(filtered)-10:]
		}

		lines := []string{}
		for _, t := range filtered {
			lines = append(lines, fmt.Sprintf("• **%s**: %s - %s (<t:%d:R>)",
				t.Type, coinDisplay(t.Amount), t.Reason, t.Timestamp/1000))
		}

		title := "Recent Transactions"
		if targetUser != nil {
			title = fmt.Sprintf("%s's Transactions", targetUser.Username)
		}

		embed := &discordgo.MessageEmbed{
			Title:       title,
			Description: strings.Join(lines, "\n"),
			Color:       ColorBlue,
			Footer:      embedFooter("💰 Economy"),
			Timestamp:   time.Now().Format(time.RFC3339),
		}
		respondEmbed(s, i, embed)

	default:
		respondError(s, i, "Error", "Unknown subcommand.")
	}
}

// Helper functions
func logTransaction(userID, txType, itemID string, amount int, grantedBy, reason string) {
	transactions := []transactionLog{}
	_ = readData("transactions.json", &transactions)

	transactions = append(transactions, transactionLog{
		UserID:    userID,
		Type:      txType,
		ItemID:    itemID,
		Amount:    amount,
		GrantedBy: grantedBy,
		Reason:    reason,
		Timestamp: time.Now().UnixMilli(),
	})

	// Keep only last 1000 transactions
	if len(transactions) > 1000 {
		transactions = transactions[len(transactions)-1000:]
	}

	_ = writeData("transactions.json", transactions)
}

func getActiveBoosts(userID string) []activeBoost {
	economyUsers := map[string]economyUser{}
	_ = readData("economy-users.json", &economyUsers)
	ecoUser := economyUsers[userID]

	// Filter expired boosts
	activeBoosts := []activeBoost{}
	now := time.Now().UnixMilli()
	for _, boost := range ecoUser.ActiveBoosts {
		if boost.ExpiresAt > now {
			activeBoosts = append(activeBoosts, boost)
		}
	}

	return activeBoosts
}

func optionFloat(opts []*discordgo.ApplicationCommandInteractionDataOption, name string, def float64) float64 {
	for _, o := range opts {
		if o.Name == name {
			if v, ok := o.Value.(float64); ok {
				return v
			}
		}
	}
	return def
}

// handleEconomyCmd is the top-level /economy group router.
func handleEconomyCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}
	sub := opts[0]
	subOpts := sub.Options

	switch sub.Name {
	case "balance":
		handleBalance(s, i)
	case "shop":
		handleShop(s, i, subOpts)
	case "inventory":
		handleInventory(s, i, subOpts)
	case "trade":
		handleTrade(s, i, subOpts)
	case "admin":
		handleEconomyAdmin(s, i, subOpts)
	case "rob":
		handleRob(s, i, subOpts)
	case "gift":
		handleGift(s, i, subOpts)
	case "leaderboard":
		handleEconomyLeaderboard(s, i)
	case "work":
		handleWork(s, i)
	case "lottery":
		handleLottery(s, i, subOpts)
	}
}

// ─── Rob ───────────────────────────────────────────────────────────────────────

func handleRob(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}

	target := optionUser(opts, "user")
	if target == nil {
		respondError(s, i, "Error", "You must specify a user to rob.")
		return
	}
	if target.Bot {
		respondError(s, i, "Error", "You cannot rob bots.")
		return
	}
	if target.ID == user.ID {
		respondError(s, i, "Error", "You cannot rob yourself.")
		return
	}

	// Check cooldown (5 minutes)
	gameMu.Lock()
	if cd, ok := robCooldowns[user.ID]; ok && time.Now().Unix() < cd {
		remaining := cd - time.Now().Unix()
		gameMu.Unlock()
		respondEmbed(s, i, createErrorEmbed("⏳ Cooldown", fmt.Sprintf("You must wait **%d:%02d** before robbing again.", remaining/60, remaining%60)))
		return
	}
	robCooldowns[user.ID] = time.Now().Unix() + 300
	gameMu.Unlock()

	targetCoins, _ := getCoins(target.ID)
	if targetCoins < 50 {
		respondEmbed(s, i, createErrorEmbed("🚫 Rob Failed", fmt.Sprintf("%s doesn't have enough coins to rob (need at least 50).", target.Username)))
		return
	}

	// 40% success rate
	if rand.Intn(100) < 40 {
		// Success: steal 10-30% of target's coins, max 500
		pct := 10 + rand.Intn(21) // 10-30
		stolen := (targetCoins * pct) / 100
		if stolen > 500 {
			stolen = 500
		}
		if stolen < 1 {
			stolen = 1
		}
		modifyCoins(target.ID, target.Username, -stolen)
		newBal := modifyCoins(user.ID, user.Username, stolen)
		respondEmbed(s, i, createSuccessEmbed("💰 Rob Successful!",
			fmt.Sprintf("You stole %s (%d%%) from %s!\nYour balance: %s", coinDisplay(stolen), pct, target.Username, coinDisplay(newBal))))
	} else {
		// Fail: pay fine of 100 coins
		fine := 100
		myCoins, _ := getCoins(user.ID)
		if myCoins < fine {
			fine = myCoins
		}
		newBal := modifyCoins(user.ID, user.Username, -fine)
		respondEmbed(s, i, createErrorEmbed("🚔 Rob Failed!",
			fmt.Sprintf("You were caught trying to rob %s!\nYou paid a fine of %s.\nYour balance: %s", target.Username, coinDisplay(fine), coinDisplay(newBal))))
	}
}

// ─── Gift ──────────────────────────────────────────────────────────────────────

func handleGift(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}

	target := optionUser(opts, "user")
	amount := int(optionInt(opts, "amount", 0))

	if target == nil {
		respondError(s, i, "Error", "You must specify a recipient.")
		return
	}
	if target.Bot {
		respondError(s, i, "Error", "You cannot gift coins to bots.")
		return
	}
	if target.ID == user.ID {
		respondError(s, i, "Error", "You cannot gift coins to yourself.")
		return
	}
	if amount <= 0 {
		respondError(s, i, "Error", "Gift amount must be positive.")
		return
	}

	myCoins, _ := getCoins(user.ID)
	if myCoins < amount {
		respondEmbed(s, i, createErrorEmbed("Insufficient Funds", fmt.Sprintf("You only have %s but tried to gift %s.", coinDisplay(myCoins), coinDisplay(amount))))
		return
	}

	modifyCoins(user.ID, user.Username, -amount)
	modifyCoins(target.ID, target.Username, amount)

	logTransaction(user.ID, "gift_sent", "", -amount, target.ID, fmt.Sprintf("Gift to %s", target.Username))
	logTransaction(target.ID, "gift_received", "", amount, user.ID, fmt.Sprintf("Gift from %s", user.Username))

	newBal, _ := getCoins(user.ID)
	respondEmbed(s, i, createSuccessEmbed("🎁 Gift Sent!",
		fmt.Sprintf("You gifted %s to %s!\nYour balance: %s", coinDisplay(amount), target.Username, coinDisplay(newBal))))
}

// ─── Leaderboard ───────────────────────────────────────────────────────────────

func handleEconomyLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
	economyUsers := map[string]economyUser{}
	_ = readData("economy-users.json", &economyUsers)

	type entry struct {
		UserID   string
		Username string
		Coins    int
	}

	entries := []entry{}
	for id, u := range economyUsers {
		entries = append(entries, entry{UserID: id, Username: u.Username, Coins: u.Coins})
	}

	sort.Slice(entries, func(a, b int) bool {
		return entries[a].Coins > entries[b].Coins
	})

	if len(entries) > 10 {
		entries = entries[:10]
	}

	if len(entries) == 0 {
		respondError(s, i, "Error", "No users in the economy yet.")
		return
	}

	medals := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	lines := []string{}
	for idx, e := range entries {
		lines = append(lines, fmt.Sprintf("%s **%s** — %s", medals[idx], e.Username, coinDisplay(e.Coins)))
	}

	embed := richEmbed(richEmbedOpts{
		Title:       "🏆 Economy Leaderboard",
		Description: strings.Join(lines, "\n"),
		Color:       ColorEconomy,
		Category:    "💰 Economy",
	})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: "📊 Stats", Value: fmt.Sprintf("%d total user(s)", len(economyUsers)), Inline: false,
	})
	respondEmbed(s, i, embed)
}

// ─── Work ──────────────────────────────────────────────────────────────────────

var workJobs = []string{
	"👨‍💻 You worked as a freelance developer and debugged 47 lines of spaghetti code.",
	"🍕 You delivered pizzas across town on a rainy night.",
	"🎨 You painted a mural for the local community center.",
	"📦 You sorted packages at the warehouse all afternoon.",
	"🧹 You deep-cleaned an office building after hours.",
	"🚗 You drove for a rideshare service during rush hour.",
	"📸 You photographed a wedding and caught the perfect shot.",
	"🎸 You busked on the street corner with your guitar.",
	"🌿 You landscaped a neighbor's garden beautifully.",
	"🐕 You walked 8 dogs simultaneously without tangling the leashes.",
	"☕ You worked a busy morning shift at the coffee shop.",
	"📚 You tutored students in math for the afternoon.",
	"🔧 You fixed a leaky faucet and unclogged a drain.",
	"🎤 You hosted a karaoke night at the local bar.",
	"🧑‍🍳 You catered a company lunch for 50 people.",
	"🏗️ You helped build a shed from scratch.",
	"💇 You gave haircuts at the barbershop all day.",
	"🎪 You performed magic tricks at a kids' birthday party.",
	"🚚 You helped someone move apartments across the city.",
	"🖥️ You set up a home network and fixed a printer.",
	"🧑‍🔬 You assisted in a chemistry lab experiment.",
	"📝 You transcribed 3 hours of meeting recordings.",
}

func handleWork(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}

	// Check cooldown (30 minutes)
	gameMu.Lock()
	if cd, ok := workCooldowns[user.ID]; ok && time.Now().Unix() < cd {
		remaining := cd - time.Now().Unix()
		gameMu.Unlock()
		respondEmbed(s, i, createErrorEmbed("⏳ Cooldown", fmt.Sprintf("You must wait **%d:%02d** before working again.", remaining/60, remaining%60)))
		return
	}
	workCooldowns[user.ID] = time.Now().Unix() + 1800
	gameMu.Unlock()

	earned := 50 + rand.Intn(151) // 50-200
	job := workJobs[rand.Intn(len(workJobs))]
	newBal := modifyCoins(user.ID, user.Username, earned)
	logTransaction(user.ID, "work", "", earned, "", "Work earnings")

	respondEmbed(s, i, createSuccessEmbed("💼 Work Complete!",
		fmt.Sprintf("%s\n\nYou earned %s!\nBalance: %s", job, coinDisplay(earned), coinDisplay(newBal))))
}

// ─── Lottery ───────────────────────────────────────────────────────────────────

func handleLottery(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}

	tickets := int(optionInt(opts, "tickets", 1))
	if tickets < 1 || tickets > 10 {
		respondError(s, i, "Error", "You can buy 1-10 tickets.")
		return
	}

	cost := tickets * 10
	myCoins, _ := getCoins(user.ID)
	if myCoins < cost {
		respondEmbed(s, i, createErrorEmbed("Insufficient Funds", fmt.Sprintf("You need %s but only have %s.", coinDisplay(cost), coinDisplay(myCoins))))
		return
	}

	modifyCoins(user.ID, user.Username, -cost)

	// Generate numbers: 3 user numbers, 3 draw numbers (1-10)
	userNums := [3]int{rand.Intn(10) + 1, rand.Intn(10) + 1, rand.Intn(10) + 1}
	drawNums := [3]int{rand.Intn(10) + 1, rand.Intn(10) + 1, rand.Intn(10) + 1}

	// Count matches
	matches := 0
	used := [3]bool{}
	for _, un := range userNums {
		for j, dn := range drawNums {
			if un == dn && !used[j] {
				matches++
				used[j] = true
				break
			}
		}
	}

	var multiplier int
	var resultText string
	switch matches {
	case 3:
		multiplier = 100
		resultText = "🎉🎉🎉 **JACKPOT!** All 3 numbers match!"
	case 2:
		multiplier = 10
		resultText = "🎉🎉 **Great!** 2 numbers match!"
	case 1:
		multiplier = 2
		resultText = "🎉 **Nice!** 1 number matches!"
	default:
		multiplier = 0
		resultText = "😞 No matches. Better luck next time!"
	}

	winnings := cost * multiplier
	var newBal int
	if winnings > 0 {
		newBal = modifyCoins(user.ID, user.Username, winnings)
	} else {
		newBal, _ = getCoins(user.ID)
	}

	embed := &discordgo.MessageEmbed{
		Title: "🎰 Lottery",
		Description: fmt.Sprintf("🎫 **Your numbers:** %d %d %d\n🎱 **Draw numbers:** %d %d %d\n\n%s",
			userNums[0], userNums[1], userNums[2],
			drawNums[0], drawNums[1], drawNums[2],
			resultText),
		Color: ColorPurple,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "🎫 Tickets", Value: fmt.Sprintf("%d (cost: %s)", tickets, coinDisplay(cost)), Inline: true},
			{Name: "💰 Winnings", Value: fmt.Sprintf("%s (%dx)", coinDisplay(winnings), multiplier), Inline: true},
			{Name: "💳 Balance", Value: coinDisplay(newBal), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    embedFooter("🎰 Lottery"),
	}
	respondEmbed(s, i, embed)
}

