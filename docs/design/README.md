# Design Notes

This folder holds website-facing design prototypes for Zeb. They are not part
of the Go CLI build.

## Terminal Launch Replica

`terminal-launch-replica.html` is a standalone browser preview of the moment a
user opens `zeb tui`. It is intentionally a terminal vignette, not a website
logo animation. Open it directly in a browser:

```bash
open docs/design/terminal-launch-replica.html
```

`TerminalLaunchReplica.tsx` is the same idea as a dependency-free React
component. It uses the same terminal glyph mark as the CLI `block-pulse` intro:

```text
          
▄▄▄▄▄▄▄▄▄ 
▀▀▀▀▀████ 
   ▄███▀  
 ▄███▀    
█████████ 
          
```

The animation has two layers:

1. A slow per-cell pulse with staggered delays.
2. A quiet terminal scan line that passes over the glyphs.

Useful CSS variables:

```css
--zeb-bg: #09090d;
--zeb-terminal: #242a33;
--zeb-terminal-deep: #1f2530;
--zeb-ink: #f3f4f6;
--zeb-muted: #8b8d98;
--zeb-accent: #ff6fd8;
--zeb-accent-2: #77d7ff;
```
