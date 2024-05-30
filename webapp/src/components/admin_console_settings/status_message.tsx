import React from 'react';

type Props = {
    state: 'warn' | 'success';
    message: string;
}

const StatusMessage = (props: Props) => {
    const {state, message} = props;

    if (state === 'warn') {
        return (
            <div>
                <div className='alert alert-warning'>
                    <i
                        className='fa fa-warning'
                        title='Warning Icon'
                    />
                    <span>{message}</span>
                </div>
            </div>
        );
    }

    return (
        <div>
            <div className='alert alert-success'>
                <i
                    className='fa fa-check'
                    title='Success Icon'
                />
                <span>{message}</span>
            </div>
        </div>
    );
};

export default StatusMessage;
