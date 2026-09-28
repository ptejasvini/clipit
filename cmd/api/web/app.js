const MAX_FILE_SIZE = 2 * 1024 * 1024;
const uploadZone = document.querySelector('.upload-zone');
const fileInput = document.querySelector('#file-input');
const pickButton = document.querySelector('#pick-button');
const gallery = document.querySelector('#gallery');
const emptyState = document.querySelector('#empty-state');
const searchInput = document.querySelector('#search-input');
const progress = document.querySelector('#upload-progress');
const progressLabel = document.querySelector('#progress-label');
const progressValue = document.querySelector('#progress-value');
const progressBar = document.querySelector('#progress-bar');
const mediaCount = document.querySelector('#media-count');
const toast = document.querySelector('#toast');
let mediaItems = [];
let toastTimer;

async function request(url, options = {}) {
  const response = await fetch(url, options);
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    throw new Error(payload.error || `Request failed (${response.status})`);
  }
  return response.status === 204 ? null : response.json();
}

function notify(message) {
  toast.textContent = message;
  toast.classList.remove('translate-y-3', 'opacity-0');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toast.classList.add('translate-y-3', 'opacity-0'), 2800);
}

function formatSize(size) {
  return size < 1024 * 1024 ? `${Math.max(1, Math.round(size / 1024))} KB` : `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function renderGallery() {
  const query = searchInput.value.trim().toLocaleLowerCase();
  const visibleItems = mediaItems.filter((item) => item.name.toLocaleLowerCase().includes(query));
  mediaCount.textContent = `${mediaItems.length} ${mediaItems.length === 1 ? 'image' : 'images'}`;
  gallery.replaceChildren();
  emptyState.classList.toggle('hidden', visibleItems.length > 0);

  if (mediaItems.length > 0 && visibleItems.length === 0) {
    document.querySelector('#empty-title').textContent = 'Nothing found';
    document.querySelector('#empty-description').textContent = 'Try another name.';
  } else if (mediaItems.length === 0) {
    document.querySelector('#empty-title').textContent = 'A fresh page';
    document.querySelector('#empty-description').textContent = 'Your uploaded images will find a home here.';
  }

  visibleItems.forEach((item, index) => {
    const card = document.createElement('article');
    card.className = 'media-card group min-w-0 overflow-hidden rounded-xl border border-ink/10 bg-white';
    card.style.animationDelay = `${Math.min(index * 35, 280)}ms`;

    const image = document.createElement('img');
    image.src = item.url;
    image.alt = item.name;
    image.loading = 'lazy';
    image.className = 'aspect-square w-full bg-paper object-cover';
    image.addEventListener('error', () => image.classList.add('hidden'), { once: true });

    const details = document.createElement('div');
    details.className = 'p-3';
    const name = document.createElement('p');
    name.className = 'truncate text-sm font-medium text-ink';
    name.title = item.name;
    name.textContent = item.name;
    const metadata = document.createElement('p');
    metadata.className = 'mt-1 text-xs text-ink/50';
    metadata.textContent = `${item.width} × ${item.height} · ${formatSize(item.size)}`;

    const actions = document.createElement('div');
    actions.className = 'mt-3 flex items-center justify-between border-t border-ink/10 pt-2';
    const download = document.createElement('a');
    download.href = item.url;
    download.download = item.name;
    download.className = 'text-xs font-medium text-forest hover:underline focus-visible:outline-2 focus-visible:outline-forest';
    download.textContent = 'Download';
    const copy = document.createElement('button');
    copy.type = 'button';
    copy.className = 'grid h-8 w-8 place-items-center rounded-md text-ink/55 hover:bg-mint hover:text-forest focus-visible:outline-2 focus-visible:outline-forest';
    copy.title = 'Copy image';
    copy.setAttribute('aria-label', `Copy ${item.name} to clipboard`);
    copy.innerHTML = '<svg viewBox="0 0 24 24" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2"/></svg>';
    copy.addEventListener('click', () => copyImage(item));
    const remove = document.createElement('button');
    remove.type = 'button';
    remove.className = 'rounded px-2 py-1 text-xs text-ink/55 hover:bg-coral/10 hover:text-coral focus-visible:outline-2 focus-visible:outline-coral';
    remove.textContent = 'Delete';
    remove.setAttribute('aria-label', `Delete ${item.name}`);
    remove.addEventListener('click', () => deleteMedia(item));

    actions.append(download, copy, remove);
    details.append(name, metadata, actions);
    card.append(image, details);
    gallery.append(card);
  });
}

async function loadMedia() {
  try {
    const result = await request('/api/media');
    mediaItems = result.items;
    renderGallery();
  } catch (error) {
    notify(error.message);
  }
}

async function withRetry(operation) {
  let lastError;
  for (let attempt = 0; attempt < 3; attempt += 1) {
    try {
      return await operation();
    } catch (error) {
      lastError = error;
      if (attempt < 2) await new Promise((resolve) => setTimeout(resolve, 350 * (attempt + 1)));
    }
  }
  throw lastError;
}

function updateProgress(file, complete, total) {
  const percentage = Math.round((complete / total) * 100);
  progress.classList.remove('hidden');
  progressLabel.textContent = file.name;
  progressValue.textContent = `${percentage}%`;
  progressBar.style.width = `${percentage}%`;
}

async function startUpload(file) {
  if (!file || !['image/png', 'image/jpeg', 'image/gif'].includes(file.type)) {
    notify('Choose a PNG, JPG or GIF image.');
    return;
  }
  if (file.size <= 0 || file.size > MAX_FILE_SIZE) {
    notify('Images must be smaller than 2 MB.');
    return;
  }

  pickButton.disabled = true;
  pickButton.classList.add('opacity-50');
  try {
    const chunkSize = 512 * 1024;
    const totalParts = Math.ceil(file.size / chunkSize);
    const saved = JSON.parse(localStorage.getItem('clipit-upload') || 'null');
    let session;
    if (saved && saved.name === file.name && saved.size === file.size) {
      try {
        session = await request(`/api/uploads/${encodeURIComponent(saved.id)}`);
        if (session.status !== 'uploading') session = null;
      } catch {
        session = null;
      }
    }
    if (!session) {
      session = await request('/api/uploads', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: file.name, size: file.size }),
      });
      localStorage.setItem('clipit-upload', JSON.stringify({ id: session.id, name: file.name, size: file.size }));
    }

    const completed = new Set(session.completedParts);
    for (let part = 0; part < totalParts; part += 1) {
      if (!completed.has(part)) {
        const chunk = file.slice(part * chunkSize, Math.min(file.size, (part + 1) * chunkSize));
        await withRetry(() => request(`/api/uploads/${encodeURIComponent(session.id)}/parts/${part}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/octet-stream' },
          body: chunk,
        }));
      }
      updateProgress(file, part + 1, totalParts);
    }
    await request(`/api/uploads/${encodeURIComponent(session.id)}/complete`, { method: 'POST' });
    localStorage.removeItem('clipit-upload');
    fileInput.value = '';
    notify('Image added to your library.');
    await loadMedia();
    setTimeout(() => progress.classList.add('hidden'), 900);
  } catch (error) {
    notify(error.message);
  } finally {
    pickButton.disabled = false;
    pickButton.classList.remove('opacity-50');
  }
}

async function deleteMedia(item) {
  if (!window.confirm(`Delete “${item.name}” from your library?`)) return;
  try {
    await request(`/api/media/${encodeURIComponent(item.id)}`, { method: 'DELETE' });
    mediaItems = mediaItems.filter((entry) => entry.id !== item.id);
    renderGallery();
    notify('Image deleted.');
  } catch (error) {
    notify(error.message);
  }
}

function copyImage(item) {
  if (!navigator.clipboard?.write || typeof ClipboardItem === 'undefined') {
    notify('Image clipboard is unavailable. Open Clipit on localhost or HTTPS in a supported browser.');
    return;
  }

  const imageData = fetch(item.url)
    .then((response) => {
      if (!response.ok) throw new Error('Could not load this image.');
      return response.blob();
    })
    .then((blob) => blob.type === 'image/png' ? blob : convertToPNG(blob));

  try {
    const clipboardWrite = navigator.clipboard.write([
      new ClipboardItem({ 'image/png': imageData }),
    ]);
    clipboardWrite.then(() => notify('Image copied. Paste it into your chat.'))
      .catch((error) => notify(error.name === 'NotAllowedError' ? 'Clipboard access was denied. Allow clipboard access and try again.' : error.message));
  } catch (error) {
    notify(error.message);
  }
}

async function convertToPNG(blob) {
  const bitmap = await createImageBitmap(blob);
  const canvas = document.createElement('canvas');
  canvas.width = bitmap.width;
  canvas.height = bitmap.height;
  const context = canvas.getContext('2d');
  if (!context) {
    bitmap.close();
    throw new Error('This browser could not prepare the image for copying.');
  }
  context.drawImage(bitmap, 0, 0);
  bitmap.close();
  return new Promise((resolve, reject) => {
    canvas.toBlob((png) => png ? resolve(png) : reject(new Error('Could not convert this image for copying.')), 'image/png');
  });
}

pickButton.addEventListener('click', () => fileInput.click());
fileInput.addEventListener('change', () => startUpload(fileInput.files[0]));
searchInput.addEventListener('input', renderGallery);
uploadZone.addEventListener('dragover', (event) => {
  event.preventDefault();
  uploadZone.classList.add('is-dragging');
});
uploadZone.addEventListener('dragleave', (event) => {
  if (!uploadZone.contains(event.relatedTarget)) uploadZone.classList.remove('is-dragging');
});
uploadZone.addEventListener('drop', (event) => {
  event.preventDefault();
  uploadZone.classList.remove('is-dragging');
  startUpload(event.dataTransfer.files[0]);
});

loadMedia();
