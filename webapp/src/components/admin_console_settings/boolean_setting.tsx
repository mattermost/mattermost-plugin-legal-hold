import {Radio} from '@mattermost/compass-ui/components/radio';

import BaseSetting from './base_setting';

type Props = {
    id: string;
    name: string;
    helpText: string;
    onChange: (value: boolean) => void;
    value: boolean;
    disabled?: boolean;
};

const BooleanSetting = (props: Props) => {
    return (
        <BaseSetting
            id={props.id}
            name={props.name}
            helpText={props.helpText}
            group={true}
        >
            <div className='legal-hold-setting__radios'>
                <Radio
                    data-testid={`${props.id}-true`}
                    name={props.id}
                    onChange={() => props.onChange(true)}
                    checked={props.value}
                    disabled={props.disabled}
                >
                    {'true'}
                </Radio>
                <Radio
                    data-testid={`${props.id}-false`}
                    name={props.id}
                    onChange={() => props.onChange(false)}
                    checked={!props.value}
                    disabled={props.disabled}
                >
                    {'false'}
                </Radio>
            </div>
        </BaseSetting>
    );
};

export default BooleanSetting;
