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

    constructor(props) {
        super(props);
        this.state = {
            email: '',
        };
    }

    resetEmail = () => {
        this.setState({email: ''});
    }

    onCancel = () => {
        if (this.props.onCancel) {
            this.props.onCancel();
        }

        this.resetEmail();
        this.props.hideConfirmationModal();
    }

    onConfirm = () => {
        if (this.props.onConfirm) {
            this.props.onConfirm(this.state.email || '');
        }

        this.resetEmail();
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
                    <p>{'You are about to send feedback about Mattermost. Are you sure?'}</p>
                    <p>{'You can optionally provide an email address if you wish to be contacted by our team.'}</p>
                    <div className='form-group'>

                        <input
                            className='form-control'
                            aria-label='Email'
                            type='email'
                            placeholder='Email (optional)'
                            value={this.state.email}
                            onChange={(e) => this.setState({email: e.target.value})}
                        />
                    </div>

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
