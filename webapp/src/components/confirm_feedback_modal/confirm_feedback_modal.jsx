import PropTypes from 'prop-types';
import React from 'react';
import {Modal} from 'react-bootstrap';

export default class ConfirmFeedbackModal extends React.PureComponent {
    static propTypes = {
        onCancel: PropTypes.func,
        onConfirm: PropTypes.func,
        hideConfirmationModal: PropTypes.func.isRequired,
        show: PropTypes.bool.isRequired,
    };

    onCancel = () => {
        if (this.props.onCancel) {
            this.props.onCancel();
        }

        this.props.hideConfirmationModal();
    }

    onConfirm = () => {
        if (this.props.onConfirm) {
            this.props.onConfirm();
        }

        this.props.hideConfirmationModal();
    }

    render() {
        return (
            <Modal
                className='modal-confirm'
                show={this.props.show}
                onHide={this.onCancel}
            >
                <Modal.Header>
                    <Modal.Title>{'Send Feedback'}</Modal.Title>
                </Modal.Header>
                <Modal.Body>
                    {'You are about to send feedback about Mattermost. Is that correct?'}
                </Modal.Body>
                <Modal.Footer>
                    <button
                        type='button'
                        className='btn btn-link btn-cancel'
                        onClick={this.onCancel}
                    >
                        {'Cancel'}
                    </button>
                    <button
                        autoFocus={true}
                        type='button'
                        className='btn btn-primary'
                        onClick={this.onConfirm}
                    >
                        {'Yes'}
                    </button>
                </Modal.Footer>
            </Modal>
        );
    }
}