import React from 'react';
import {Modal} from 'react-bootstrap';

interface Props {
    onHide: () => void;
    onConfirm: () => void;
    show: boolean;
}

const RunConfirmationModal = ({show, onHide, onConfirm}: Props) => {
    return (
        <Modal
            show={show}
            onHide={onHide}
            centered={true}
        >
            <Modal.Header closeButton={true}>
                <Modal.Title>{'Confirm Legal Hold Run'}</Modal.Title>
            </Modal.Header>
            <Modal.Body>
                {'Are you sure you want to run this legal hold now?'}
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
                    {'Run Now'}
                </button>
            </Modal.Footer>
        </Modal>
    );
};

export default RunConfirmationModal;
