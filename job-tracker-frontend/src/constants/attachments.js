// Attachment document kinds, kept in step with allowedFileTypes in
// backend/internal/service/job.go.
export const FILE_TYPES = [
  { value: 'resume', label: 'Resume' },
  { value: 'cover_letter', label: 'Cover Letter' },
  { value: 'cover_letter_typed', label: 'Cover Letter (Typed)' },
  { value: 'question_responses', label: 'Question Responses' },
  { value: 'interview_prep', label: 'Interview Prep' },
];

// Falls back to the raw value so an unrecognised type from the API still reads
// as something rather than being mislabelled.
export const fileTypeLabel = (value) =>
  FILE_TYPES.find((type) => type.value === value)?.label || value;

// What the browser can render itself. DOC/DOCX are absent on purpose: nothing
// renders them without a converter, so they stay download-only. Older rows may
// predate mime_type, so fall back to the extension.
const VIEWABLE = {
  'text/html': { icon: '🌐', ext: /\.html?$/i },
  'application/pdf': { icon: '📕', ext: /\.pdf$/i },
  'text/plain': { icon: '📃', ext: /\.txt$/i },
  'text/markdown': { icon: '📃', ext: /\.(md|markdown)$/i },
};

const viewableEntry = (attachment) => {
  if (!attachment) return null;
  const byMime = VIEWABLE[attachment.mime_type];
  if (byMime) return byMime;
  const name = attachment.file_name || '';
  return Object.values(VIEWABLE).find((entry) => entry.ext.test(name)) || null;
};

export const isViewable = (attachment) => viewableEntry(attachment) !== null;

export const attachmentIcon = (attachment) => viewableEntry(attachment)?.icon || '📄';
