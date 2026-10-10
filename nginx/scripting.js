/**
 * Return the installed njs version and JavaScript engine over HTTP.
 * @param r The current HTTP request.
 */
function version(r) {
    r.return(200, 'njs ' + njs.version + ' engine ' + njs.engine + '\n');
}

/**
 * Return the installed njs version and engine for a stream return directive.
 * @param s The current stream session.
 * @returns A greeting terminated by a newline.
 */
function greeting(s) {
    return 'njs ' + njs.version + ' engine ' + njs.engine + '\n';
}

/**
 * Expose HTTP and stream handlers for the scripting example.
 */
export default {version, greeting};
