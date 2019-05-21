import {createSelector} from 'reselect';

import {Preferences} from 'mattermost-redux/constants';
import {get} from 'mattermost-redux/selectors/entities/preferences';
import {getMyTeams} from 'mattermost-redux/selectors/entities/teams';

import {id as pluginId} from './manifest';

const MESSAGE_DISPLAY = 'message_display';
const MESSAGE_DISPLAY_DEFAULT = 'clean';
const MESSAGE_DISPLAY_COMPACT = 'compact';

function getPluginState(state) {
    return state['plugins-' + pluginId] || {};
}

export function getConfirmationModalState(state) {
    return getPluginState(state).confirmationModal || {};
}

export function getWindowWidth(state) {
    return getPluginState(state).windowWidth;
}

export function isSurveyPostSmall(state, isRHS) {
    return isRHS || !useFullSurveyPost(state);
}

function isCompactView(state) {
    const compactView = get(state, Preferences.CATEGORY_DISPLAY_SETTINGS, MESSAGE_DISPLAY, MESSAGE_DISPLAY_DEFAULT) === MESSAGE_DISPLAY_COMPACT;
    return compactView;
}

function isSidebarOpen(state) {
    return state.views.rhs.isSidebarOpen;
}

function isTeamSidebarVisible(state) {
    return getMyTeams(state).length > 1;
}

// The required width for the regular and small survey posts with different combinations of display settings
const requiredWidths = [
    {compactView: false, sidebarOpen: false, fullWidth: 856, smallWidth: 769},
    {compactView: false, sidebarOpen: true, fullWidth: 1260, smallWidth: 1090},
    {compactView: true, sidebarOpen: false, fullWidth: 898, smallWidth: 769},
    {compactView: true, sidebarOpen: true, fullWidth: 1298, smallWidth: 1130},
];

export const useSurveyPost = createSelector(
    isCompactView,
    isSidebarOpen,
    isTeamSidebarVisible,
    getWindowWidth,
    (compactView, sidebarOpen, teamSidebarVisible, windowWidth) => {
        let {smallWidth} = requiredWidths.find((requirements) => requirements.compactView === compactView && requirements.sidebarOpen === sidebarOpen);

        if (!teamSidebarVisible) {
            // Remove the extra space needed for the team sidebar
            smallWidth -= 65;
        }

        return windowWidth >= smallWidth && windowWidth > 768;
    }
);

export const useFullSurveyPost = createSelector(
    isCompactView,
    isSidebarOpen,
    isTeamSidebarVisible,
    getWindowWidth,
    (compactView, sidebarOpen, teamSidebarVisible, windowWidth) => {
        let {fullWidth} = requiredWidths.find((requirements) => requirements.compactView === compactView && requirements.sidebarOpen === sidebarOpen);

        if (!teamSidebarVisible) {
            // Remove the extra space needed for the team sidebar
            fullWidth -= 65;
        }

        return windowWidth >= fullWidth && windowWidth > 768;
    }
);
