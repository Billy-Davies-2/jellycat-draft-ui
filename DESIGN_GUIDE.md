# Jellycat Draft UI - Design Guide

## Overview
The Jellycat Draft UI has been redesigned to reflect the soft, cuddly aesthetic of the Jellycat brand while incorporating subtle football-themed elements to create a unique and inviting user experience.

## Design Philosophy

### Jellycat-Inspired Aesthetic
The design draws inspiration from Jellycat's signature style:
- **Soft & Rounded**: Generous border-radius values (1rem to 1.5rem) create a plush, friendly feel
- **Pastel Colors**: Gentle, easy-on-the-eyes color palette with soft pinks, purples, and blues
- **Warm Tones**: Creamy whites and peachy beiges provide warmth and comfort
- **Gentle Animations**: Smooth, slow animations that don't distract but add playfulness

### Football Theme Integration
Football elements are integrated subtly to complement the Jellycat aesthetic:
- **Pastel Grass**: Soft green gradients (#C8E6C9 to #A5D6A7) for field-inspired accents
- **Decorative Icons**: Football (⚽), trophy (🏆), and goalpost emojis used sparingly
- **Team Badges**: Football field-inspired backgrounds for team elements

## Color Palette

### Primary Jellycat Colors
```css
--color-jellycat-cream: #FFF9F0;      /* Warm white background */
--color-jellycat-beige: #F5EDE0;      /* Soft beige */
--color-jellycat-soft-pink: #FFE5EC;  /* Primary pink */
--color-jellycat-blush: #FFB5C5;      /* Accent pink */
--color-jellycat-lavender: #E8D5F2;   /* Primary purple */
--color-jellycat-lilac: #D5B8E0;      /* Accent purple */
--color-jellycat-sky: #E0F2FE;        /* Soft blue */
--color-jellycat-mint: #D5F5E3;       /* Gentle mint */
--color-jellycat-peach: #FFE4D6;      /* Warm peach */
```

### Football-Themed Colors
```css
--color-football-grass: #C8E6C9;      /* Light grass green */
--color-football-field: #A5D6A7;      /* Field green */
--color-football-orange: #FFCC80;     /* Accent orange */
```

### Background Gradients
```css
/* Main background */
background: linear-gradient(135deg, #FFE5EC 0%, #E8D5F2 50%, #E0F2FE 100%);

/* Button gradient */
background: linear-gradient(to right, #f472b6, #a855f7);

/* Football field gradient */
background: linear-gradient(to bottom, #C8E6C9, #A5D6A7);
```

## Typography

### Font Families
- **Display/Headings**: 'Quicksand' - Rounded, friendly, perfect for titles
- **Body Text**: 'Nunito' - Clean, readable, warm sans-serif
- **Fallback**: System fonts (-apple-system, BlinkMacSystemFont, etc.)

### Usage
```css
.font-display {
  font-family: 'Quicksand', 'Nunito', sans-serif;
}

body {
  font-family: 'Nunito', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}
```

## Custom Components

### Buttons

#### Jellycat Button (`.btn-jellycat`)
Primary action button with pink-to-purple gradient:
```html
<button class="btn-jellycat">Draft Jellycat</button>
```
- **Colors**: Pink (#f472b6) to Purple (#a855f7)
- **Hover**: Darker gradient with scale effect (1.05)
- **Active**: Scale down (0.95) for tactile feedback
- **Shadow**: Soft pink-tinted shadow

#### Football Button (`.btn-football`)
Secondary button with grass gradient:
```html
<button class="btn-football">Join Team</button>
```
- **Colors**: Light grass (#C8E6C9) to dark grass (#A5D6A7)
- **Hover**: Shadow enhancement with scale effect

### Cards

#### Jellycat Card (`.card-jellycat`)
Primary card component for content containers:
```html
<div class="card-jellycat">
  <!-- Content -->
</div>
```
- **Border**: 2px solid pink (#fbcfe8)
- **Radius**: 1rem (16px)
- **Shadow**: Soft pink-tinted shadow
- **Hover**: Enhanced shadow and border color change

#### Player Card (`.player-card`)
Special card for player selection with rotation on hover:
```html
<div class="player-card">
  <!-- Player info -->
</div>
```
- **Hover Effects**: Scale (1.05) + Rotate (-1deg)
- **Transition**: All properties in 0.3s

### Form Elements

#### Input Fields (`.input-jellycat`)
Styled text inputs and selects:
```html
<input type="text" class="input-jellycat" placeholder="Team name...">
```
- **Border**: 2px solid pink (#fbcfe8)
- **Focus**: Border color change + subtle ring shadow
- **Padding**: 0.75rem 1rem for comfortable touch targets

### Tabs

#### Tab Navigation (`.tab-jellycat`)
```html
<button class="tab-jellycat active">Draft Board</button>
```
- **Default**: Transparent with hover gradient
- **Active**: Pink-purple gradient + bottom border (4px)
- **Rounded**: Top corners only (0.75rem)

### Badges

#### Cuddle Points Badge (`.cuddle-badge`)
For displaying player points:
```html
<div class="cuddle-badge">150 Points</div>
```
- **Gradient**: Yellow (#fef3c7) to Pink (#fbcfe8)
- **Color**: Purple text (#7c3aed)
- **Size**: Small (0.875rem font)

#### Team Badge (`.team-badge`)
For team-related indicators:
```html
<div class="team-badge">Team Name</div>
```
- **Gradient**: Football field colors
- **Color**: Dark green text (#2e7d32)

## Animations

### Float Animation
Gentle up-and-down movement for decorative elements:
```css
.animate-float {
  animation: float 3s ease-in-out infinite;
}

.animate-float-slow {
  animation: float 4s ease-in-out infinite;
}
```

### Bounce Slow
Slower version of Tailwind's bounce:
```css
.animate-bounce-slow {
  animation: bounce 3s infinite;
}
```

### Pulse Soft
Gentle pulsing for attention:
```css
.animate-pulse-soft {
  animation: pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}
```

## Shadows

### Soft Shadows
Pink-tinted shadows for depth without harshness:

```css
.shadow-soft {
  box-shadow: 0 2px 15px -3px rgba(255, 182, 193, 0.3), 
              0 4px 6px -2px rgba(255, 182, 193, 0.15);
}

.shadow-soft-lg {
  box-shadow: 0 10px 30px -5px rgba(255, 182, 193, 0.4), 
              0 8px 10px -5px rgba(255, 182, 193, 0.2);
}

.shadow-soft-xl {
  box-shadow: 0 20px 40px -10px rgba(255, 182, 193, 0.5), 
              0 15px 20px -10px rgba(255, 182, 193, 0.3);
}
```

### Football Shadow
Green-tinted shadow for football elements:
```css
.shadow-football {
  box-shadow: 0 4px 20px -2px rgba(168, 214, 167, 0.4);
}
```

## Text Gradients

### Jellycat Gradient Text
```html
<h1 class="text-gradient-jellycat">Jellycat Fantasy Draft</h1>
```
- **Gradient**: Pink → Purple → Blue
- **Effect**: Cheerful, friendly headline treatment

### Football Gradient Text
```html
<span class="text-gradient-football">Score!</span>
```
- **Gradient**: Dark green → Light green
- **Effect**: Football field-inspired text

## Responsive Design

The design uses Tailwind's responsive breakpoints:

```css
/* Mobile first */
- Base styles apply to all sizes

/* Tablet (md: 48rem / 768px) */
.md\:grid-cols-2 { ... }

/* Desktop (lg: 64rem / 1024px) */
.lg\:col-span-3 { ... }

/* Large desktop (xl: 80rem / 1280px) */
.xl\:grid-cols-4 { ... }
```

## Custom Scrollbar

Themed scrollbar for webkit browsers:
```css
::-webkit-scrollbar {
  width: 10px;
  height: 10px;
}

::-webkit-scrollbar-track {
  background: #fce7f3; /* Pink-50 */
  border-radius: 9999px;
}

::-webkit-scrollbar-thumb {
  background: linear-gradient(to bottom, #f9a8d4, #e9d5ff);
  border-radius: 9999px;
}
```

## Decorative Elements

### Floating Emojis
Decorative background elements add playfulness:
```html
<!-- In base.html body -->
<div class="fixed top-10 right-10 text-6xl opacity-20 animate-float-slow pointer-events-none">⚽</div>
<div class="fixed bottom-20 left-20 text-5xl opacity-20 animate-bounce-slow pointer-events-none">🧸</div>
<div class="fixed top-1/3 left-10 text-4xl opacity-15 animate-float pointer-events-none">🏈</div>
```

## Building the Styles

To rebuild the CSS after making changes:

```bash
# Download Tailwind CSS CLI (first time only)
curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
chmod +x tailwindcss-linux-x64

# Build the styles
./tailwindcss-linux-x64 -i static/css/input.css -o static/css/styles.css --minify
```

## Best Practices

1. **Maintain Consistency**: Use the defined color variables and component classes
2. **Soft Transitions**: Keep all transitions smooth (300ms or longer)
3. **Generous Spacing**: Use padding/margins liberally for a spacious feel
4. **Rounded Corners**: Prefer large border-radius values (1rem+)
5. **Gentle Animations**: Avoid jarring or fast animations
6. **Accessibility**: Maintain sufficient color contrast ratios
7. **Mobile First**: Design for mobile, enhance for desktop

## Accessibility Considerations

- All interactive elements have sufficient touch targets (min 44x44px)
- Color contrast meets WCAG AA standards
- Focus states are clearly visible
- Animations respect `prefers-reduced-motion`
- Semantic HTML structure maintained

## Future Enhancements

Potential additions to the design system:
- Dark mode variant with deep purples and blues
- More football-themed decorative patterns
- Loading states with plush toy animations
- Celebration animations for successful drafts
- Sound effects (optional) for interactions
