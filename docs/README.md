# Discorbo Landing Page

This is the GitHub Pages landing site for Discorbo.

## 🌐 Viewing the Site

### Locally
Open `index.html` in your browser or use a local server:
```bash
# Python 3
python -m http.server 8000

# Node.js (if you have http-server installed)
npx http-server

# Then visit: http://localhost:8000
```

### On GitHub Pages

1. **Enable GitHub Pages:**
   - Go to your repository settings
   - Navigate to "Pages" section
   - Under "Source", select the `main` branch
   - Under "Folder", select `/docs`
   - Click "Save"

2. **Your site will be live at:**
   ```
   https://[your-username].github.io/Discorbo/
   ```

## 📁 Files

- `index.html` - Main landing page
- `styles.css` - All styling and animations
- `script.js` - Interactive features and animations
- `README.md` - This file

## ✨ Features

- **Fully Responsive** - Works on all devices
- **Modern Design** - Discord-themed with gradients and animations
- **Interactive Elements** - Smooth scrolling, hover effects, scroll animations
- **SEO Optimized** - Meta tags and Open Graph support
- **Fast Loading** - Optimized CSS and minimal dependencies
- **Accessible** - Semantic HTML and keyboard navigation

## 🎨 Customization

### Colors
Edit the CSS variables in `styles.css`:
```css
:root {
    --primary: #5865f2;      /* Discord Blurple */
    --secondary: #eb459e;    /* Pink accent */
    --success: #57f287;      /* Green */
    /* ... more colors */
}
```

### Content
Edit `index.html` to update:
- Hero section text
- Feature descriptions
- Command lists
- Footer links

### Invite Link
The invite link is used in multiple places. Update it throughout `index.html`:
```
https://discord.com/api/oauth2/authorize?client_id=YOUR_CLIENT_ID&permissions=277025770560&scope=bot%20applications.commands
```

## 🚀 Deploy Updates

After making changes:
1. Commit your changes
2. Push to GitHub
3. GitHub Pages will automatically rebuild (takes 1-2 minutes)

## 📱 Browser Support

- Chrome/Edge (latest)
- Firefox (latest)
- Safari (latest)
- Mobile browsers (iOS Safari, Chrome Mobile)

## 🎁 Easter Eggs

- Try the Konami code: ↑↑↓↓←→←→BA
- Smooth animations on scroll
- Interactive stat counters

---

**Made with ❤️ by Sachin Shankar**
