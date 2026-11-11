/**
 * Return the installed njs version and JavaScript engine over HTTP.
 * @param {NginxHTTPRequest} r The current HTTP request.
 */
const version = function version(r) {
  r.return(200, `njs ${njs.version} engine ${njs.engine}\n`);
};

/**
 * Return the installed njs version and engine for a stream return directive.
 * @param {NginxStreamSession} _s The current stream session.
 * @returns {string} A greeting terminated by a newline.
 */
const greeting = function greeting(_s) {
  return `njs ${njs.version} engine ${njs.engine}\n`;
};

/**
 * Expose HTTP and stream handlers for the scripting example.
 */
export default { greeting, version };
