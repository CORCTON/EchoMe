export async function uploadFile(file: File): Promise<string> {
  const formData = new FormData();
  formData.append('file', file);

  const res = await fetch('/api/upload', { method: 'POST', body: formData });
  const json = await res.json().catch(() => ({}));
  if (!res.ok || !json.url) {
    throw new Error('Upload failed');
  }
  return json.url as string;
}

export default uploadFile;