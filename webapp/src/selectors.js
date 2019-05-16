import {id as pluginId} from './manifest';

function getPluginState(state) {
    return state['plugins-' + pluginId] || {};
}

export function getConfirmationModalState(state) {
    return getPluginState(state).confirmationModal || {};
}

export function isScorePostSmall(state, isRHS) {
    // A score post should use the smaller design if it's in the RHS or on a small screen with the RHS open
    return isRHS || (getPluginState(state).windowWidth < 1280 && state.views.rhs.isSidebarOpen);
}
