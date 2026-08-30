// Attachment document kinds, kept in step with allowedFileTypes in
// backend/internal/service/job.go.
export const FILE_TYPES = [
  { value: 'resume', label: 'Resume' },
  { value: 'cover_letter', label: 'Cover Letter' },
  { value: 'cover_letter_typed', label: 'Cover Letter (Typed)' },
  { value: 'question_responses', label: 'Question Responses' },
];

// Falls back to the raw value so an unrecognised type from the API still reads
// as something rather than being mislabelled.
export const fileTypeLabel = (value) =>
  FILE_TYPES.find((type) => type.value === value)?.label || value;
