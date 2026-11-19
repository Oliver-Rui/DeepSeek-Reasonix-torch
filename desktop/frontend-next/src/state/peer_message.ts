// A message another session sent arrives wrapped in the block the model read it
// in. The reader wants what the other session said, not the frame that told the
// model whose words these were. Content written before the block existed has
// none; then the whole line is the body, which is what the reader would have
// seen anyway.
const PEER_MESSAGE = /<peer-message\b[^>]*>([\s\S]*?)<\/peer-message>/;

export function peerMessageBody(content: string): string {
  const found = PEER_MESSAGE.exec(content);
  return (found ? found[1] : content).trim();
}
