package integration

type basicInfoInitialiser struct{}

func (b *basicInfoInitialiser) Apply(r *Run) error {
	err := b.setupContext(r)
	if err != nil {
		return err
	}
	return nil
}

func (b *basicInfoInitialiser) setupContext(r *Run) error {
	obj, errMake := makeObject(r.VM)
	if errMake != nil {
		return errMake
	}

	if err := obj.Set("run_id", r.ID); err != nil {
		return err
	}
	if err := obj.Set("run_reason", r.StartContext.TriggerKind); err != nil {
		return err
	}
	if err := obj.Set("trigger_id", r.StartContext.TriggerUID); err != nil {
		return err
	}
	if err := obj.Set("start_time", r.Started); err != nil {
		return err
	}

	return r.VM.Set("context", obj)
}
