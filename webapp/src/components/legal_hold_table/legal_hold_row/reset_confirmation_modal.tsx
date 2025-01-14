import React from 'react';
import {Modal} from 'react-bootstrap';

interface Props {
    show: boolean;
    onHide: () => void;
    onConfirm: () => void;
}

const ResetConfirmationModal = ({show, onHide, onConfirm}: Props) => {
    return (
        <Modal
            show={show}
            onHide={onHide}
        >
            <Modal.Header closeButton={true}>
                <Modal.Title>{'Are you sure?'}</Modal.Title>
            </Modal.Header>
            <Modal.Body>
                {'This will modify the legal hold status forcefully setting the status to \'not running\'. Only run this if you know what you are doing.'}
            </Modal.Body>
            <Modal.Footer>
                <button
                    type='button'
                    className='btn btn-link'
                    onClick={onHide}
                >
                    {'Cancel'}
                </button>
                <button
                    type='button'
                    className='btn btn-primary'
                    onClick={onConfirm}
                >
                    {'Confirm'}
                </button>
            </Modal.Footer>
        </Modal>
    );
};

export default ResetConfirmationModal;
