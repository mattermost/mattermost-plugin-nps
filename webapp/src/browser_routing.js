const WebappUtils = window.WebappUtils;

const navigateToUrl = (urlPath) => {
    WebappUtils.browserHistory.push(urlPath);
};

export const navigateToChannel = (team, channel) => {
    navigateToUrl(`${team}/messages/@${channel}`);
};
