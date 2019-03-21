import {id as pluginId} from './manifest';

function getPluginState(state) {
    return state['plugins-' + pluginId] || {};
}

export function getConfirmationModalState(state) {
    return getPluginState(state).confirmationModal || {};
}