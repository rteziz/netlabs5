'use strict';
(() => {
 const canvas = document.getElementById('background');
 const ctx = canvas.getContext('2d');
 if (!ctx) return;
 const spacing = 32, stepInterval = 1000, moveDuration = 800, maxStep = 3;
 const strokeWidth = 3, previousStrokeWidth = 1.5, shapeCount = 9;
 const spiralHueSweep = 240;
 const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
 const toggle = document.getElementById('motion-toggle');
 let symbols = [], width = 0, height = 0, columns = 0, rows = 0;
 let frame = 0, lastStep = performance.now(), paused = reducedMotion.matches;
 const randomStep = () => Math.floor(Math.random() * (maxStep * 2 + 1)) - maxStep;
 const clamp = (value, max) => Math.max(0, Math.min(max - 1, value));
 function paint(now) {
  ctx.clearRect(0, 0, width, height);
  const progress = paused ? 1 : Math.min(1, Math.max(0, (now - lastStep) / moveDuration));
  const ease = progress * progress * (3 - 2 * progress);
  for (const symbol of symbols) {
   const x = (symbol.fromX + (symbol.x - symbol.fromX) * ease + .5) * spacing;
   const y = (symbol.fromY + (symbol.y - symbol.fromY) * ease + .5) * spacing;
   const rotation = symbol.fromRotation + (symbol.rotation - symbol.fromRotation) * ease;
   ctx.save(); ctx.translate(x, y); ctx.rotate(rotation);
   ctx.strokeStyle = symbol.color; ctx.fillStyle = symbol.color;
   ctx.lineWidth = strokeWidth; ctx.lineCap = 'round'; ctx.lineJoin = 'round'; ctx.beginPath();
   // Thicken inward so the outer footprint stays the same.
   const radius = symbol.radius - (strokeWidth - previousStrokeWidth) / 2;
   switch (symbol.kind) {
    case 0: ctx.arc(0, 0, radius, 0, Math.PI * 2); ctx.stroke(); break;
    case 1:
     ctx.moveTo(-radius, 0); ctx.lineTo(radius, 0);
     ctx.moveTo(0, -radius); ctx.lineTo(0, radius); ctx.stroke(); break;
    case 2:
     for (let i = 0; i <= 64; i++) {
      const angle = i / 64 * Math.PI * 4, r = i / 64 * radius;
      const px = Math.cos(angle) * r, py = Math.sin(angle) * r;
      if (!i) ctx.moveTo(px, py); else ctx.lineTo(px, py);
     }
     ctx.stroke(); break;
    case 3:
     ctx.arc(0, 0, radius, 0, Math.PI * 2); ctx.stroke();
     ctx.beginPath(); ctx.arc(0, 0, 2.5, 0, Math.PI * 2); ctx.fill(); break;
    case 4: ctx.arc(0, 0, symbol.radius * .45, 0, Math.PI * 2); ctx.fill(); break;
    case 5: {
     const side = radius / Math.SQRT2;
     ctx.rect(-side, -side, side * 2, side * 2); ctx.stroke(); break;
    }
    case 6:
     ctx.moveTo(0, -radius); ctx.lineTo(radius, 0);
     ctx.lineTo(0, radius); ctx.lineTo(-radius, 0);
     ctx.closePath(); ctx.stroke(); break;
    case 7:
    case 8: {
     const sides = symbol.kind === 7 ? 3 : 6;
     for (let i = 0; i < sides; i++) {
      const angle = i * Math.PI * 2 / sides - Math.PI / 2;
      const px = Math.cos(angle) * radius, py = Math.sin(angle) * radius;
      if (!i) ctx.moveTo(px, py); else ctx.lineTo(px, py);
     }
     ctx.closePath(); ctx.stroke(); break;
    }
   }
   ctx.restore();
  }
 }
 function tick(now) {
  if (now - lastStep >= stepInterval) {
   for (const symbol of symbols) {
    symbol.fromX = symbol.x; symbol.fromY = symbol.y;
    symbol.fromRotation = symbol.rotation % (Math.PI * 2);
    symbol.rotation = symbol.fromRotation + Math.floor(Math.random() * 5) * Math.PI / 4;
    // Move to another grid position. Collisions are intentional: order becomes chaos.
    const step = randomStep() || 1;
    const horizontal = Math.random() < .5;
    const dx = horizontal ? step : 0, dy = horizontal ? 0 : step;
    symbol.x = clamp(symbol.x + dx, columns);
    symbol.y = clamp(symbol.y + dy, rows);
   }
   lastStep = now;
  }
  paint(now); frame = requestAnimationFrame(tick);
 }
 function sync() {
  cancelAnimationFrame(frame);
  toggle.dataset.i18n = paused ? 'resumeMotion' : 'pauseMotion';
  toggle.textContent = document.documentElement.lang === 'en'
   ? (paused ? 'Animate background' : 'Pause background')
   : (paused ? 'Включить фон' : 'Пауза фона');
  toggle.setAttribute('aria-pressed', String(paused));
  lastStep = performance.now();
  if (!paused && !document.hidden) frame = requestAnimationFrame(tick);
  else paint(lastStep + moveDuration);
 }
 function resize() {
  width = window.innerWidth; height = window.innerHeight;
  const ratio = Math.min(window.devicePixelRatio || 1, 2);
  canvas.width = Math.round(width * ratio); canvas.height = Math.round(height * ratio);
  ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
  columns = Math.ceil(width / spacing); rows = Math.ceil(height / spacing);
  const spiralRadius = Math.max(spacing, Math.hypot(width, height) / 2);
  symbols = [];
  for (let y = 0; y < rows; y++) for (let x = 0; x < columns; x++) {
   const dx = (x + .5) * spacing - width / 2, dy = (y + .5) * spacing - height / 2;
   // Continuous full-spectrum hue; scale the spiral to the viewport for gentler transitions.
   const phase = Math.atan2(dy, dx) + Math.hypot(dx, dy) / spiralRadius * spiralHueSweep * Math.PI / 180;
   const hue = (phase * 180 / Math.PI + 360) % 360;
   const lightness = 49 + 5 * Math.sin(phase);
   symbols.push({x, y, fromX:x, fromY:y, kind:Math.floor(Math.random()*shapeCount),
    radius:9 + Math.random()*7, rotation:0, fromRotation:0,
    color:`hsla(${hue}, 65%, ${lightness}%, 0.55)`});
  }
  sync();
 }
 toggle.addEventListener('click', () => { paused = !paused; sync(); });
 reducedMotion.addEventListener('change', () => { paused = reducedMotion.matches; sync(); });
 document.addEventListener('visibilitychange', sync);
 window.addEventListener('resize', resize);
 window.addEventListener('pagehide', () => cancelAnimationFrame(frame));
 window.addEventListener('pageshow', sync);
 resize();
})();
