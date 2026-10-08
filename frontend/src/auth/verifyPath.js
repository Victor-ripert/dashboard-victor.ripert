// URL of the page where a user enters the code emailed to them.
// The email is in the URL (not router state) so the page survives a reload.
export function verifyPath(email) {
  return `/verify?email=${encodeURIComponent(email)}`;
}
