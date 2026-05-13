const messages = document.querySelector('#messages');
const form = document.querySelector('#chat-form');
const question = document.querySelector('#question');

function addMessage(role, text) {
  const article = document.createElement('article');
  article.className = role;
  article.textContent = text;
  messages.appendChild(article);
  messages.scrollTop = messages.scrollHeight;
}

form.addEventListener('submit', async (event) => {
  event.preventDefault();
  const text = question.value.trim();
  if (!text) return;
  question.value = '';
  addMessage('user', text);
  const response = await fetch('/api/chat', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({question: text})});
  const payload = await response.json();
  addMessage('doctor', payload.answer || payload.error || 'No answer');
});