const WebappUtils = window.WebappUtils;

const navigateToUrl = (urlPath) => {
    WebappUtils.browserHistory.push(urlPath);
};

export const navigateToChannel = (team, channelName) => {
    const teamPrefix = team.startsWith('/') ? team.slice(1) : team;
    navigateToUrl(`/${teamPrefix}/channels/${channelName}`);
};
