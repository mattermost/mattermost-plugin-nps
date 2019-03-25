import {combineReducers} from 'redux';

import * as ActionTypes from './action_types';

function confirmationModal(state = {show: false}, action) {
    switch (action.type) {
    case ActionTypes.SHOW_CONFIRMATION_MODAL:
        return {
            show: true,
            onConfirm: action.onConfirm,
            onCancel: action.onCancel,
        };
    case ActionTypes.HIDE_CONFIRMATION_MODAL:
        return {
            show: false,
            onConfirm: null,
            onCancel: null,
        };

    default:
        return state;
    }
}

export default combineReducers({
    confirmationModal,
});