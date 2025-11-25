const commitAnalyzer = require('@semantic-release/commit-analyzer');

const VALID_RELEASE_TYPES = new Set(['major', 'minor', 'patch', 'prerelease', 'premajor', 'preminor', 'prepatch']);

module.exports = {
  analyzeCommits: async (pluginConfig, context) => {
    const override = (process.env.RELEASE_TYPE_OVERRIDE || '').trim().toLowerCase();

    if (override && override !== 'auto') {
      if (override === 'none') {
        context.logger.log('Manual release verification requested skipping publish step.');
        return false;
      }

      if (VALID_RELEASE_TYPES.has(override)) {
        context.logger.log('Manual release override forcing a %s release.', override);
        return override;
      }

      context.logger.warn('Unknown RELEASE_TYPE_OVERRIDE "%s" ignored. Falling back to automatic detection.', override);
    }

    return commitAnalyzer.analyzeCommits(pluginConfig, context);
  },
};
