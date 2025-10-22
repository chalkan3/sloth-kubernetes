// 🦥 Sloth Kubernetes Custom JavaScript

document.addEventListener('DOMContentLoaded', function() {
  // Add sloth emoji animation to specific elements
  const slothEmojis = document.querySelectorAll('span:contains("🦥")');
  slothEmojis.forEach(emoji => {
    emoji.classList.add('sloth-emoji');
  });

  // Smooth scroll to anchors
  document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
      e.preventDefault();
      const target = document.querySelector(this.getAttribute('href'));
      if (target) {
        target.scrollIntoView({
          behavior: 'smooth',
          block: 'start'
        });
      }
    });
  });

  // Add copy button feedback
  document.querySelectorAll('.md-clipboard').forEach(button => {
    button.addEventListener('click', function() {
      const originalText = button.getAttribute('title');
      button.setAttribute('title', '🦥 Copied!');
      setTimeout(() => {
        button.setAttribute('title', originalText);
      }, 2000);
    });
  });

  // Animate sloth on page load
  console.log('%c🦥 Sloth Kubernetes Documentation', 'font-size: 20px; color: #8B4513; font-weight: bold;');
  console.log('%cSlowly, but surely... the docs are loaded!', 'font-size: 14px; color: #D2691E;');

  // Add "back to top" button behavior
  const backToTop = document.querySelector('.md-top');
  if (backToTop) {
    backToTop.addEventListener('click', function() {
      window.scrollTo({
        top: 0,
        behavior: 'smooth'
      });
    });
  }

  // Highlight external links
  document.querySelectorAll('a[href^="http"]').forEach(link => {
    if (!link.hostname.includes(window.location.hostname)) {
      link.setAttribute('target', '_blank');
      link.setAttribute('rel', 'noopener noreferrer');
      link.innerHTML += ' <span style="font-size: 0.8em;">↗</span>';
    }
  });

  // Add progress indicator for long pages
  const progressBar = document.createElement('div');
  progressBar.style.cssText = `
    position: fixed;
    top: 0;
    left: 0;
    width: 0%;
    height: 3px;
    background: linear-gradient(90deg, #8B4513, #FF8C00);
    z-index: 9999;
    transition: width 0.2s ease;
  `;
  document.body.appendChild(progressBar);

  window.addEventListener('scroll', () => {
    const windowHeight = window.innerHeight;
    const documentHeight = document.documentElement.scrollHeight;
    const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
    const scrollPercentage = (scrollTop / (documentHeight - windowHeight)) * 100;
    progressBar.style.width = scrollPercentage + '%';
  });

  // Easter egg: Konami code for sloth animation
  const konamiCode = ['ArrowUp', 'ArrowUp', 'ArrowDown', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'ArrowLeft', 'ArrowRight', 'b', 'a'];
  let konamiIndex = 0;

  document.addEventListener('keydown', (e) => {
    if (e.key === konamiCode[konamiIndex]) {
      konamiIndex++;
      if (konamiIndex === konamiCode.length) {
        activateSlothMode();
        konamiIndex = 0;
      }
    } else {
      konamiIndex = 0;
    }
  });

  function activateSlothMode() {
    const slothParty = document.createElement('div');
    slothParty.innerHTML = '🦥'.repeat(50);
    slothParty.style.cssText = `
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      font-size: 3em;
      z-index: 99999;
      pointer-events: none;
      animation: sloth-rain 5s linear;
    `;
    document.body.appendChild(slothParty);

    const style = document.createElement('style');
    style.textContent = `
      @keyframes sloth-rain {
        from { transform: translateY(-100%); opacity: 1; }
        to { transform: translateY(100vh); opacity: 0; }
      }
    `;
    document.head.appendChild(style);

    setTimeout(() => {
      slothParty.remove();
      style.remove();
    }, 5000);

    console.log('%c🦥🦥🦥 SLOTH MODE ACTIVATED! 🦥🦥🦥', 'font-size: 30px; color: #FF8C00; font-weight: bold; text-shadow: 2px 2px 4px #8B4513;');
  }
});
